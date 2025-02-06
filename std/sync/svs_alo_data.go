package sync

import (
	"fmt"
	"slices"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	spec_svs "github.com/named-data/ndnd/std/ndn/svs/v3"
)

type svsDataState struct {
	// Known is the fetched sequence number.
	// Data is handed off to outgoing pipe.
	Known uint64
	// Latest is the latest sequence number.
	// (Latest-Pending) is not in the pipeline.
	Latest uint64
	// Pending is the pending sequence number.
	// (Pending-Known) has outstanding Interests.
	Pending uint64
	// PendingData is the fetched data not yet delivered.
	PendingData map[uint64]SvsPub
	// SnapBlock is the snapshot block flag.
	// If non-zero, fetching is blocked.
	SnapBlock int
}

// svsPubOut is the pipe struct. It is necessary to bundle together differet
// kinds of output here because go does not support union types.
type svsPubOut struct {
	// Fields below are set for a normal pub
	hash string
	pub  SvsPub
	subs []func(SvsPub)

	// Fields below are set if it is a snapshot
	// Only pub will be set for a snapshot
	snapstate *spec_svs.StateVector
}

func (s *SvsALO) objectName(node enc.Name, boot uint64, seq uint64) enc.Name {
	return node.
		Append(s.opts.Svs.GroupPrefix...).
		Append(enc.NewTimestampComponent(boot)).
		Append(enc.NewSequenceNumComponent(seq)).
		WithVersion(enc.VersionImmutable)
}

func (s *SvsALO) produceObject(content enc.Wire) (enc.Name, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// This instance owns the underlying SVS instance.
	// So we can be sure that the sequence number does not
	// change while we hold the lock on this instance.
	node := s.opts.Name
	boot := s.svs.GetBootTime()
	seq := s.svs.GetSeqNo(node) + 1

	name, err := s.client.Produce(ndn.ProduceArgs{
		Name:    s.objectName(node, boot, seq),
		Content: content,
	})
	if err != nil {
		return nil, err
	}

	// We don't get notified of changes to our own state.
	// So we need to update the state vector ourselves.
	hash := node.TlvStr()
	s.state.Set(hash, boot, svsDataState{
		Known:   seq,
		Latest:  seq,
		Pending: seq,
	})

	// Inform the snapshot strategy
	s.opts.Snapshot.check(snapCheckArgs{s.state, node, hash})

	// Update the state vector
	if got := s.svs.IncrSeqNo(node); got != seq {
		panic("[BUG] sequence number mismatch - who changed it?")
	}

	return name, nil
}

// consumeCheck looks for new objects to fetch and queues them.
func (s *SvsALO) consumeCheck(node enc.Name, hash string) {
	if !s.nodePs.HasSub(node) {
		return
	}

	// Check with the snapshot strategy
	s.opts.Snapshot.check(snapCheckArgs{s.state, node, hash})

	totalPending := uint64(0)

	for _, entry := range s.state[hash] {
		fstate := entry.Value
		totalPending += fstate.Pending - fstate.Known
		if fstate.SnapBlock != 0 {
			continue
		}

		// Check if there is something to fetch
		for fstate.Pending < fstate.Latest {
			// Too many pending interests
			if totalPending > s.opts.MaxPipelineSize {
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
			s.state.Set(hash, entry.Boot, fstate)
			s.consumeObject(node, entry.Boot, fstate.Pending)
		}
	}
}

// consumeObject fetches a single object from the network.
// The callback puts data on the outgoing pipeline and re-calls
// check to fetch more objects if necessary.
func (s *SvsALO) consumeObject(node enc.Name, boot uint64, seq uint64) {
	name := s.objectName(node, boot, seq)
	s.client.Consume(name, func(status ndn.ConsumeState) {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		// Get the state vector entry
		hash := node.TlvStr()
		entry := s.state.Get(hash, boot)

		// Always check if we can fetch more
		defer s.consumeCheck(node, hash)

		// Check if this is already delivered
		if seq <= entry.Known {
			return
		}

		// Reset pending if we cancel the fetch
		resetPending := func() {
			entry.Pending = min(entry.Pending, seq-1)
			s.state.Set(hash, boot, entry)
		}

		// Check snapshot block
		if entry.SnapBlock != 0 {
			resetPending()
			return
		}

		// Check for errors
		if err := status.Error(); err != nil {
			// Check if we are still subscribed
			if !s.nodePs.HasSub(node) {
				resetPending()
				return
			}

			// Propagate the error to application
			s.errpipe <- &ErrSync{node, err}

			// TODO: exponential backoff
			time.AfterFunc(2*time.Second, func() {
				s.consumeObject(node, boot, seq)
			})
			return
		}

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
			BootTime:  boot,
			SeqNum:    seq,
		}

		// Collect all data to deliver
		var deliver []SvsPub = nil
		for {
			// Check if the next seq is available
			nextSeq := entry.Known + 1
			pub, ok := entry.PendingData[nextSeq]
			if !ok {
				break
			}

			// Even if we are no longer subscribed,
			// it's okay to delete all the pending data.
			delete(entry.PendingData, nextSeq)
			deliver = append(deliver, pub)

			// Only updating the local copy of the state
			// The state will be updated only after finding some subs.
			entry.Known = nextSeq
		}

		// Deliver the data with unlocked mutex
		if len(deliver) > 0 {
			// We got these subs when trying to deliver. It doesn't matter
			// if they were unsubscribed after this function exits. The application
			// needs to handle the callbacks correctly.
			subs := slices.Collect(s.nodePs.Subs(node))
			if len(subs) == 0 {
				return // no longer subscribed
			}

			// we WILL deliver it, update known state
			s.state.Set(hash, boot, entry)
			for _, pub := range deliver {
				s.outpipe <- svsPubOut{hash: hash, pub: pub, subs: subs}
			}
		}
	})
}

// snapshotCallback is called by the snapshot strategy to indicate
// that a snapshot has been fetched.
func (s *SvsALO) snapshotCallback(callback snapRecvCallback) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Get the snapshot under the lock, and let the strategy
	// mutate the state safely.
	snapPub, err := callback(s.state)

	// Trigger fetch for all affected publishers even if
	// the callback has failed
	defer func() {
		for name := range s.state.Iter() {
			if snapPub.Publisher.IsPrefix(name) {
				s.consumeCheck(name, name.TlvStr())
			}
		}
	}()

	if err != nil {
		s.errpipe <- &ErrSync{
			name: snapPub.Publisher,
			err:  fmt.Errorf("%w: %w", ErrSnapshot, err),
		}
		return
	}

	// Update delivered vector after the snapshot is delivered
	// (this is the reason it needs to go out on the outpipe)
	s.outpipe <- svsPubOut{
		pub:       snapPub,
		subs:      slices.Collect(s.nodePs.Subs(snapPub.Publisher)), // suspicious
		snapstate: s.state.Encode(func(state svsDataState) uint64 { return state.Known }),
	}
}
