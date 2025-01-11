package sync

import (
	"errors"
	"math"
	rand "math/rand/v2"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	spec "github.com/named-data/ndnd/std/ndn/spec_2022"
	spec_svs "github.com/named-data/ndnd/std/ndn/svs_2024"
	sec "github.com/named-data/ndnd/std/security"
	"github.com/named-data/ndnd/std/utils"
)

type SvSync struct {
	engine      ndn.Engine
	groupPrefix enc.Name
	onUpdate    func(SvSyncUpdate)

	running atomic.Bool
	stop    chan struct{}
	ticker  *time.Ticker

	periodicTimeout   time.Duration
	suppressionPeriod time.Duration

	mutex sync.Mutex
	state map[uint64]uint64
	names map[uint64]enc.Name
	mtime map[uint64]time.Time

	suppress bool
	merge    map[uint64]uint64

	recvSv chan *spec_svs.StateVector
}

type SvSyncUpdate struct {
	Name enc.Name
	High uint64
	Low  uint64
}

func NewSvSync(
	engine ndn.Engine,
	groupPrefix enc.Name,
	onUpdate func(SvSyncUpdate),
) *SvSync {
	return &SvSync{
		engine:      engine,
		groupPrefix: groupPrefix.Clone(),
		onUpdate:    onUpdate,

		running: atomic.Bool{},
		stop:    make(chan struct{}),
		ticker:  time.NewTicker(1 * time.Second),

		periodicTimeout:   30 * time.Second,
		suppressionPeriod: 200 * time.Millisecond,

		mutex: sync.Mutex{},
		state: make(map[uint64]uint64),
		names: make(map[uint64]enc.Name),
		mtime: make(map[uint64]time.Time),

		suppress: false,
		merge:    make(map[uint64]uint64),

		recvSv: make(chan *spec_svs.StateVector, 128),
	}
}

func (s *SvSync) Start() (err error) {
	err = s.engine.AttachHandler(s.groupPrefix, func(args ndn.InterestHandlerArgs) {
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
	defer s.engine.DetachHandler(s.groupPrefix)

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

func (s *SvSync) Stop() {
	s.ticker.Stop()
	s.stop <- struct{}{}
	close(s.stop)
}

func (s *SvSync) SetSeqNo(name enc.Name, seqNo uint64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	hash := s.hashName(name)
	prev := s.state[hash]

	if seqNo <= prev {
		return errors.New("SvSync: seqNo must be greater than previous")
	}

	// [Spec] When the node generates a new publication,
	// immediately emit a Sync Interest
	s.state[hash] = seqNo
	go s.sendSyncInterest()

	return nil
}

func (s *SvSync) GetSeqNo(name enc.Name) uint64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	hash := s.hashName(name)
	return s.state[hash]
}

func (s *SvSync) IncrSeqNo(name enc.Name) uint64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	hash := s.hashName(name)
	val := s.state[hash] + 1
	s.state[hash] = val

	// [Spec] When the node generates a new publication,
	// immediately emit a Sync Interest
	go s.sendSyncInterest()

	return val
}

func (s *SvSync) hashName(name enc.Name) uint64 {
	hash := name.Hash()
	if _, ok := s.names[hash]; !ok {
		s.names[hash] = name.Clone()
	}
	return hash
}

func (s *SvSync) onReceiveStateVector(sv *spec_svs.StateVector) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	isOutdated := false
	canDrop := true
	recvSet := make(map[uint64]bool)

	for _, entry := range sv.Entries {
		hash := s.hashName(entry.Name)
		recvSet[hash] = true

		prev := s.state[hash]
		if entry.SeqNo > prev {
			// [Spec] If the incoming state vector is newer,
			// update the local state vector.
			s.state[hash] = entry.SeqNo

			// [Spec] Store the current timestamp as the last update
			// time for each updated node.
			s.mtime[hash] = time.Now()

			// Notify the application of the update
			s.onUpdate(SvSyncUpdate{
				Name: entry.Name,
				High: entry.SeqNo,
				Low:  prev + 1,
			})
		} else if entry.SeqNo < prev {
			isOutdated = true

			// [Spec] If every node with an outdated sequence number
			// in the incoming state vector was updated in the last
			// SuppressionPeriod, drop the Sync Interest.
			if time.Now().After(s.mtime[hash].Add(s.suppressionPeriod)) {
				canDrop = false
			}
		}

		// [Spec] Suppression state
		if s.suppress {
			// [Spec] For every incoming Sync Interest, aggregate
			// the state vector into a MergedStateVector.
			if entry.SeqNo > s.merge[hash] {
				s.merge[hash] = entry.SeqNo
			}
		}
	}

	// The above checks each node in the incoming state vector, but
	// does not check if a node is missing from the incoming state vector.
	if !isOutdated {
		for nameHash := range s.state {
			if _, ok := recvSet[nameHash]; !ok {
				isOutdated = true
				canDrop = false
				break
			}
		}
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
	s.merge = make(map[uint64]uint64)

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
		send := false
		for nameHash, seqNo := range s.state {
			if seqNo > s.merge[nameHash] {
				send = true
				break
			}
		}
		if !send {
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
	svWire := func() enc.Wire {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		// [Spec*] Sending always triggers Steady State
		s.enterSteadyState()

		return s.encodeSv()
	}()

	// SVS v3 Sync Data
	syncName := s.groupPrefix.Append(enc.NewVersionComponent(3))

	// TODO: sign the sync data
	signer := sec.NewSha256Signer()

	dataCfg := &ndn.DataConfig{
		ContentType: utils.IdPtr(ndn.ContentTypeBlob),
	}
	data, err := s.engine.Spec().MakeData(syncName, dataCfg, svWire, signer)
	if err != nil {
		log.Errorf("SvSync: sendSyncInterest failed make data: %+v", err)
		return
	}

	// Make SVS Sync Interest
	intCfg := &ndn.InterestConfig{
		Lifetime: utils.IdPtr(1 * time.Second),
		Nonce:    utils.ConvertNonce(s.engine.Timer().Nonce()),
	}
	interest, err := s.engine.Spec().MakeInterest(syncName, intCfg, data.Wire, nil)
	if err != nil {
		log.Errorf("SvSync: sendSyncInterest failed make interest: %+v", err)
		return
	}

	// [Spec] Sync Ack Policy - Do not acknowledge Sync Interests
	err = s.engine.Express(interest, nil)
	if err != nil {
		log.Errorf("SvSync: sendSyncInterest failed express: %+v", err)
	}
}

func (s *SvSync) onSyncInterest(interest ndn.Interest) {
	if !s.running.Load() {
		return
	}

	// Check if app param is present
	if interest.AppParam() == nil {
		log.Debug("SvSync: onSyncInterest no AppParam, ignoring")
		return
	}

	// Decode Sync Data
	pkt, _, err := spec.ReadPacket(enc.NewWireReader(interest.AppParam()))
	if err != nil {
		log.Warnf("SvSync: onSyncInterest failed to parse SyncData: %+v", err)
		return
	}
	if pkt.Data == nil {
		log.Warnf("SvSync: onSyncInterest no Data, ignoring")
		return
	}

	// TODO: verify signature on Sync Data

	// Decode state vector
	svWire := pkt.Data.Content().Join()
	params, err := spec_svs.ParseSvsData(enc.NewBufferReader(svWire), false)
	if err != nil || params.StateVector == nil {
		log.Warnf("SvSync: onSyncInterest failed to parse StateVec: %+v", err)
		return
	}

	s.recvSv <- params.StateVector
}

// Call with mutex locked
func (s *SvSync) encodeSv() enc.Wire {
	entries := make([]*spec_svs.StateVectorEntry, 0, len(s.state))
	for nameHash, seqNo := range s.state {
		entries = append(entries, &spec_svs.StateVectorEntry{
			Name:  s.names[nameHash],
			SeqNo: seqNo,
		})
	}

	// Sort entries by in the NDN canonical order
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name.Compare(entries[j].Name) < 0
	})

	params := spec_svs.SvsData{
		StateVector: &spec_svs.StateVector{Entries: entries},
	}

	return params.Encode()
}

// Call with mutex locked
func (s *SvSync) enterSteadyState() {
	s.suppress = false
	// [Spec] Steady state: Reset Sync Interest timer to PeriodicTimeout
	s.ticker.Reset(s.getPeriodicTimeout())
}

func (s *SvSync) getPeriodicTimeout() time.Duration {
	// [Spec] ±10% uniform jitter
	jitter := s.periodicTimeout / 10
	min := s.periodicTimeout - jitter
	max := s.periodicTimeout + jitter
	return time.Duration(rand.Int64N(int64(max-min))) + min
}

func (s *SvSync) getSuppressionTimeout() time.Duration {
	// [Spec] Exponential decay function
	// [Spec] c = SuppressionPeriod  // constant factor
	// [Spec] v = random(0, c)       // uniform random value
	// [Spec] f = 10.0               // decay factor
	c := float64(s.suppressionPeriod)
	v := float64(rand.Int64N(int64(s.suppressionPeriod)))
	f := float64(10.0)

	// [Spec] SuppressionTimeout = c * (1 - e^((v - c) / (c / f)))
	timeout := time.Duration(c * (1 - math.Pow(math.E, ((v-c)/(c/f)))))

	return timeout
}
