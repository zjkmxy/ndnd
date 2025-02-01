package sync

import (
	gosync "sync"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
)

// Max pending interests for a producer.
const SvsAloMaxPending = 20

// SvsALO is a Sync Transport with At Least One delivery semantics.
type SvsALO struct {
	// opts is the configuration options.
	opts SvsAloOpts
	// svs is the underlying Sync Transport.
	svs *SvSync
	// client is the object client.
	client ndn.Client

	// mutex protects the instance.
	mutex gosync.Mutex
	// wmutex protects the write (produce) state.
	wmutex gosync.Mutex

	// state is the current state.
	state SvMap[SeqFetchState]
	// outpipe is the channel for delivering data.
	outpipe chan SvsPub
	// stop is the stop signal.
	stop chan struct{}
}

type SeqFetchState struct {
	// Known is the fetched state vector.
	// Data is already delivered to the application.
	Known uint64
	// Latest is the latest state vector.
	// (Latest-Pending) is not in the pipeline.
	Latest uint64
	// Pending is the pending state vector.
	// (Pending-Known) has outstanding Interests.
	Pending uint64
	// PendingData is the fetched data not yet delivered.
	PendingData map[uint64]SvsPub
}

type SvsAloOpts struct {
	// Name is the name of this instance producer.
	Name enc.Name
	// Svs is the options for the underlying SVS instance.
	Svs SvSyncOpts
	// OnData is the callback for new data.
	OnData func(SvsPub)
}

// NewSvsALO creates a new SvsALO instance.
func NewSvsALO(opts SvsAloOpts) *SvsALO {
	if len(opts.Name) == 0 {
		panic("Name is required")
	}
	if opts.OnData == nil {
		panic("OnData is required")
	}

	s := &SvsALO{
		opts:    opts,
		svs:     nil,
		client:  opts.Svs.Client,
		state:   NewSvMap[SeqFetchState](0),
		outpipe: make(chan SvsPub, 256),
		stop:    make(chan struct{}),
	}

	s.opts.Svs.OnUpdate = s.onUpdate
	s.svs = NewSvSync(s.opts.Svs)

	return s
}

// String is the log identifier.
func (s *SvsALO) String() string {
	return "svs-alo"
}

// Start starts the SvsALO instance.
func (s *SvsALO) Start() {
	s.svs.Start()
	go func() {
		defer s.svs.Stop()
		for {
			select {
			case <-s.stop:
				return
			case data := <-s.outpipe:
				s.opts.OnData(data)
			}
		}
	}()
}

// Stop stops the SvsALO instance.
func (s *SvsALO) Stop() {
	s.stop <- struct{}{}
	close(s.stop)
}

// Publish sends a message to the group
func (s *SvsALO) Publish(content enc.Wire) (enc.Name, error) {
	s.wmutex.Lock()
	defer s.wmutex.Unlock()

	// This instance owns the underlying SVS instance.
	// So we can be sure that the sequence number does not
	// change while we hold the lock on this instance.
	boot := s.svs.GetBootTime()
	seq := s.svs.GetSeqNo(s.opts.Name) + 1

	name, err := s.client.Produce(ndn.ProduceArgs{
		Name:    s.objectName(s.opts.Name, boot, seq),
		Content: content,
	})
	if err != nil {
		return nil, err
	}

	// Update the state vector
	if got := s.svs.IncrSeqNo(s.opts.Name); got != seq {
		panic("[BUG] sequence number mismatch - who changed it?")
	}

	return name, nil
}

// onUpdate is the handler for new sequence numbers.
func (s *SvsALO) onUpdate(update SvSyncUpdate) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Update the latest known state.
	hash := update.Name.String()
	entry := s.state.Get(hash, update.Boot)
	entry.Latest = update.High
	s.state.Set(hash, update.Boot, entry)

	// TODO: notify app for subscription changes

	// Check if we want to queue new fetch for this update.
	s.check(update.Name, hash)
}

// _check looks for new work once
// returns true to continue the check
func (s *SvsALO) check(node enc.Name, hash string) {
	// TODO: check if subscribed to this node, otherwise exit

	totalPending := uint64(0)

	for _, entry := range s.state[hash] {
		fstate := entry.value
		totalPending += fstate.Pending - fstate.Known

		// Check if there is something to fetch
		for fstate.Pending < fstate.Latest {
			// Too many pending interests
			if totalPending > SvsAloMaxPending {
				return
			}

			// Skip fetching if the client is congested and there are
			// at least two pending fetches for this producer.
			if totalPending >= 2 && s.client.IsCongested() {
				return
			}

			// Queue the fetch
			totalPending++
			fstate.Pending++
			s.state.Set(hash, entry.boot, fstate)
			s.consumeObject(node, entry.boot, fstate.Pending)
		}
	}
}

func (s *SvsALO) objectName(node enc.Name, boot uint64, seq uint64) enc.Name {
	return node.
		Append(s.opts.Svs.GroupPrefix...).
		Append(enc.NewTimestampComponent(boot)).
		Append(enc.NewSequenceNumComponent(seq)).
		WithVersion(enc.VersionImmutable)
}

func (s *SvsALO) consumeObject(node enc.Name, boot uint64, seq uint64) {
	log.Debug(s, "Consume", "node", node, "boot", boot, "seq", seq)

	name := s.objectName(node, boot, seq)
	s.client.Consume(name, func(status ndn.ConsumeState) {
		if !status.IsComplete() {
			return
		}

		s.mutex.Lock()
		defer s.mutex.Unlock()

		// TODO: check if still subscribed, otherwise discard

		if err := status.Error(); err != nil {
			log.Warn(s, err.Error(), "node", node) // TODO: remove
			time.AfterFunc(2*time.Second, func() {
				s.consumeObject(node, boot, seq)
			})
			return
		}

		hash := node.String()
		entry := s.state.Get(hash, boot)
		if entry.PendingData == nil {
			entry.PendingData = make(map[uint64]SvsPub)
			s.state.Set(hash, boot, entry)
		}

		// Store the content for in-order delivery
		// The size of this map is upper bounded
		entry.PendingData[seq] = SvsPub{
			Publisher: node,
			Content:   status.Content(),
			DataName:  name,
		}

		// Check if we can deliver the data
		for {
			// Check if the next seq is available
			nextSeq := entry.Known + 1
			pub, ok := entry.PendingData[nextSeq]
			if !ok {
				break
			}
			delete(entry.PendingData, nextSeq)
			entry.Known = nextSeq
			s.state.Set(hash, boot, entry)
			s.outpipe <- pub
		}

		// Check if we can fetch more
		s.check(node, hash)
	})
}
