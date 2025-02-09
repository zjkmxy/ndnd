package sync

import (
	"fmt"
	"math"
	rand "math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	spec "github.com/named-data/ndnd/std/ndn/spec_2022"
	spec_svs "github.com/named-data/ndnd/std/ndn/svs/v3"
	"github.com/named-data/ndnd/std/types/optional"
	"github.com/named-data/ndnd/std/utils"
)

type SvSync struct {
	o SvSyncOpts

	running atomic.Bool
	stop    chan struct{}
	ticker  *time.Ticker

	mutex sync.Mutex
	state SvMap[uint64]
	mtime map[string]time.Time

	suppress bool
	merge    SvMap[uint64]

	recvSv chan *spec_svs.StateVector
}

type SvSyncOpts struct {
	Client      ndn.Client
	GroupPrefix enc.Name
	OnUpdate    func(SvSyncUpdate)

	InitialState      *spec_svs.StateVector
	BootTime          uint64
	PeriodicTimeout   time.Duration
	SuppressionPeriod time.Duration
}

type SvSyncUpdate struct {
	Name enc.Name
	Boot uint64
	High uint64
	Low  uint64
}

// NewSvSync creates a new SV Sync instance.
func NewSvSync(opts SvSyncOpts) *SvSync {
	// Check required options
	if opts.Client == nil {
		panic("SvSync: Client is required")
	}
	if len(opts.GroupPrefix) == 0 {
		panic("SvSync: GroupPrefix is required")
	}
	if opts.OnUpdate == nil {
		panic("SvSync: OnUpdate is required")
	}

	// Use initial state if provided
	initialState := NewSvMap[uint64](0)
	if opts.InitialState != nil {
		for _, node := range opts.InitialState.Entries {
			hash := node.Name.TlvStr()
			for _, entry := range node.SeqNoEntries {
				initialState.Set(hash, entry.BootstrapTime, entry.SeqNo)
			}
		}
	}

	// Set default options
	if opts.BootTime == 0 {
		opts.BootTime = uint64(time.Now().Unix())
	}
	if opts.PeriodicTimeout == 0 {
		opts.PeriodicTimeout = 30 * time.Second
	}
	if opts.SuppressionPeriod == 0 {
		opts.SuppressionPeriod = 200 * time.Millisecond
	}

	// Deep copy referenced options
	opts.GroupPrefix = opts.GroupPrefix.Clone()

	return &SvSync{
		o: opts,

		running: atomic.Bool{},
		stop:    make(chan struct{}),
		ticker:  time.NewTicker(1 * time.Second),

		mutex: sync.Mutex{},
		state: initialState,
		mtime: make(map[string]time.Time),

		suppress: false,
		merge:    NewSvMap[uint64](0),

		recvSv: make(chan *spec_svs.StateVector, 128),
	}
}

// Instance log identifier
func (s *SvSync) String() string {
	return fmt.Sprintf("svs (%s)", s.o.GroupPrefix)
}

// Start the SV Sync instance.
func (s *SvSync) Start() (err error) {
	err = s.o.Client.Engine().AttachHandler(s.o.GroupPrefix, func(args ndn.InterestHandlerArgs) {
		go s.onSyncInterest(args.Interest)
	})
	if err != nil {
		return err
	}

	go s.main()
	go s.sendSyncInterest()

	return nil
}

func (s *SvSync) main() {
	defer s.o.Client.Engine().DetachHandler(s.o.GroupPrefix)

	s.running.Store(true)
	defer s.running.Store(false)

	for {
		select {
		case <-s.ticker.C:
			s.timerExpired()
		case sv := <-s.recvSv:
			s.onReceiveStateVector(sv)
		case <-s.stop:
			return
		}
	}
}

// Stop the SV Sync instance.
func (s *SvSync) Stop() {
	s.ticker.Stop()
	s.stop <- struct{}{}
	close(s.stop)
}

// GetSeqNo returns the sequence number for a name.
func (s *SvSync) GetSeqNo(name enc.Name) uint64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.state.Get(name.TlvStr(), s.o.BootTime)
}

// SetSeqNo sets the sequence number for a name.
// The instance must only set sequence numbers for names it owns.
// The sequence number must be greater than the previous value.
func (s *SvSync) SetSeqNo(name enc.Name, seqNo uint64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	hash := name.TlvStr()

	entry := s.state.Get(hash, s.o.BootTime)
	if seqNo <= entry {
		return fmt.Errorf("SvSync: seqNo must be greater than previous")
	}

	// [Spec] When the node generates a new publication,
	// immediately emit a Sync Interest
	s.state.Set(hash, s.o.BootTime, seqNo)
	go s.sendSyncInterest()

	return nil
}

// IncrSeqNo increments the sequence number for a name.
// The instance must only increment sequence numbers for names it owns.
func (s *SvSync) IncrSeqNo(name enc.Name) uint64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	hash := name.TlvStr()
	entry := s.state.Get(hash, s.o.BootTime)
	entry++
	s.state.Set(hash, s.o.BootTime, entry)

	// [Spec] When the node generates a new publication,
	// immediately emit a Sync Interest
	go s.sendSyncInterest()

	return entry
}

func (s *SvSync) GetBootTime() uint64 {
	return s.o.BootTime
}

func (s *SvSync) onReceiveStateVector(sv *spec_svs.StateVector) {
	// Deliver the updates after this call is done
	// This ensures the mutex is not held during the callback
	// This function, in turn, runs on our main goroutine, so
	// that ensures the updates are delivered in order.
	var updates []SvSyncUpdate
	defer func() {
		for _, update := range updates {
			s.o.OnUpdate(update)
		}
	}()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	isOutdated := false
	canDrop := true
	recvSv := NewSvMap[uint64](len(sv.Entries))

	for _, node := range sv.Entries {
		hash := node.Name.TlvStr()

		// Walk through the state vector entries in reverse order.
		// The ordering is important so we deliver the newest boot time first.
		for seqi := range node.SeqNoEntries {
			entry := node.SeqNoEntries[len(node.SeqNoEntries)-1-seqi]
			recvSv.Set(hash, entry.BootstrapTime, entry.SeqNo)

			// [SPEC] If any received BootstrapTime is more than 86400s in the
			// future compared to current time, the entire state vector SHOULD be ignored.
			if entry.BootstrapTime > uint64(time.Now().Unix())+86400 {
				log.Warn(s, "Dropping state vector with far future BootstrapTime: %d", entry.BootstrapTime)
				return
			}

			// Get existing state vector entry
			known := s.state.Get(hash, entry.BootstrapTime)
			if entry.SeqNo > known {
				// [Spec] If the incoming state vector is newer,
				// update the local state vector.
				s.state.Set(hash, entry.BootstrapTime, entry.SeqNo)

				// [Spec] Store the current timestamp as the last update
				// time for each updated node.
				s.mtime[hash] = time.Now()

				// Notify the application of the update
				updates = append(updates, SvSyncUpdate{
					Name: node.Name,
					Boot: entry.BootstrapTime,
					High: entry.SeqNo,
					Low:  known + 1,
				})
			} else if entry.SeqNo < known {
				isOutdated = true

				// [Spec] If every node with an outdated sequence number
				// in the incoming state vector was updated in the last
				// SuppressionPeriod, drop the Sync Interest.
				if time.Now().After(s.mtime[hash].Add(s.o.SuppressionPeriod)) {
					canDrop = false
				}
			}

			// [Spec] Suppression state
			if s.suppress {
				// [Spec] For every incoming Sync Interest, aggregate
				// the state vector into a MergedStateVector.
				known := s.merge.Get(hash, entry.BootstrapTime)
				if entry.SeqNo > known {
					s.merge.Set(hash, entry.BootstrapTime, entry.SeqNo)
				}
			}
		}
	}

	// The above checks each node in the incoming state vector, but
	// does not check if a node is missing from the incoming state vector.
	if !isOutdated && s.state.IsNewerThan(recvSv, func(_, _ uint64) bool { return false }) {
		isOutdated = true
		canDrop = false
	}

	if !isOutdated {
		// [Spec] Suppression state: Move to Steady State.
		// [Spec] Steady state: Reset Sync Interest timer.
		s.enterSteadyState()
		return
	} else if canDrop || s.suppress {
		// See above for explanation
		return
	}

	// [Spec] Incoming Sync Interest is outdated.
	// [Spec] Move to Suppression State.
	s.suppress = true
	s.merge = make(SvMap[uint64], len(s.state))

	// [Spec] When entering Suppression State, reset
	// the Sync Interest timer to SuppressionTimeout
	s.ticker.Reset(s.getSuppressionTimeout())
}

func (s *SvSync) timerExpired() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// [Spec] Suppression State
	if s.suppress {
		// [Spec] If MergedStateVector is up-to-date; no inconsistency.
		if !s.state.IsNewerThan(s.merge, func(a, b uint64) bool { return a > b }) {
			s.enterSteadyState()
			return
		}
		// [Spec] If MergedStateVector is outdated; inconsistent state.
		// Emit up-to-date Sync Interest.
	}

	// [Spec] On expiration of timer emit a Sync Interest
	// with the current local state vector.
	go s.sendSyncInterest()
}

func (s *SvSync) sendSyncInterest() {
	if !s.running.Load() {
		return
	}

	// Critical section
	sv := func() *spec_svs.StateVector {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		// [Spec*] Sending always triggers Steady State
		s.enterSteadyState()

		return s.state.Encode(func(s uint64) uint64 { return s })
	}()
	svWire := (&spec_svs.SvsData{StateVector: sv}).Encode()

	// SVS v3 Sync Data
	syncName := s.o.GroupPrefix.Append(enc.NewVersionComponent(3))

	// Sign Sync Data
	signer := s.o.Client.SuggestSigner(syncName)
	if signer == nil {
		log.Error(s, "SvSync failed to find valid signer", "name", syncName)
		return
	}

	dataCfg := &ndn.DataConfig{
		ContentType: optional.Some(ndn.ContentTypeBlob),
	}
	data, err := s.o.Client.Engine().Spec().MakeData(syncName, dataCfg, svWire, signer)
	if err != nil {
		log.Error(s, "sendSyncInterest failed make data", "err", err)
		return
	}

	// Make SVS Sync Interest
	intCfg := &ndn.InterestConfig{
		Lifetime: optional.Some(1 * time.Second),
		Nonce:    utils.ConvertNonce(s.o.Client.Engine().Timer().Nonce()),
	}
	interest, err := s.o.Client.Engine().Spec().MakeInterest(syncName, intCfg, data.Wire, nil)
	if err != nil {
		log.Error(s, "sendSyncInterest failed make interest", "err", err)
		return
	}

	// [Spec] Sync Ack Policy - Do not acknowledge Sync Interests
	err = s.o.Client.Engine().Express(interest, nil)
	if err != nil {
		log.Error(s, "sendSyncInterest failed express", "err", err)
	}
}

func (s *SvSync) onSyncInterest(interest ndn.Interest) {
	if !s.running.Load() {
		return
	}

	// Check if app param is present
	if interest.AppParam() == nil {
		log.Debug(s, "onSyncInterest no AppParam, ignoring")
		return
	}

	// Decode Sync Data
	data, sigCov, err := spec.Spec{}.ReadData(enc.NewWireView(interest.AppParam()))
	if err != nil {
		log.Warn(s, "onSyncInterest failed to parse SyncData", "err", err)
		return
	}

	// Validate signature
	s.o.Client.Validate(data, sigCov, func(valid bool, err error) {
		if !valid || err != nil {
			log.Warn(s, "SvSync failed to validate signature", "name", data.Name(), "valid", valid, "err", err)
			return
		}

		// Decode state vector
		svWire := data.Content().Join()
		params, err := spec_svs.ParseSvsData(enc.NewBufferView(svWire), false)
		if err != nil || params.StateVector == nil {
			log.Warn(s, "onSyncInterest failed to parse StateVec", "err", err)
			return
		}

		s.recvSv <- params.StateVector
	})
}

// Call with mutex locked
func (s *SvSync) enterSteadyState() {
	s.suppress = false
	// [Spec] Steady state: Reset Sync Interest timer to PeriodicTimeout
	s.ticker.Reset(s.getPeriodicTimeout())
}

func (s *SvSync) getPeriodicTimeout() time.Duration {
	// [Spec] ±10% uniform jitter
	jitter := s.o.PeriodicTimeout / 10
	min := s.o.PeriodicTimeout - jitter
	max := s.o.PeriodicTimeout + jitter
	return time.Duration(rand.Int64N(int64(max-min))) + min
}

func (s *SvSync) getSuppressionTimeout() time.Duration {
	// [Spec] Exponential decay function
	// [Spec] c = SuppressionPeriod  // constant factor
	// [Spec] v = random(0, c)       // uniform random value
	// [Spec] f = 10.0               // decay factor
	c := float64(s.o.SuppressionPeriod)
	v := float64(rand.Int64N(int64(s.o.SuppressionPeriod)))
	f := float64(10.0)

	// [Spec] SuppressionTimeout = c * (1 - e^((v - c) / (c / f)))
	timeout := time.Duration(c * (1 - math.Pow(math.E, ((v-c)/(c/f)))))

	return timeout
}
