package sync

import (
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
)

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

func (s *SvsALO) objectName(node enc.Name, boot uint64, seq uint64) enc.Name {
	return node.
		Append(s.opts.Svs.GroupPrefix...).
		Append(enc.NewTimestampComponent(boot)).
		Append(enc.NewSequenceNumComponent(seq)).
		WithVersion(enc.VersionImmutable)
}

func (s *SvsALO) produceObject(content enc.Wire) (enc.Name, error) {
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

// consumeCheck looks for new objects to fetch and queues them.
func (s *SvsALO) consumeCheck(node enc.Name, hash string) {
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

func (s *SvsALO) consumeObject(node enc.Name, boot uint64, seq uint64) {
	name := s.objectName(node, boot, seq)
	s.client.Consume(name, func(status ndn.ConsumeState) {
		if !status.IsComplete() {
			return
		}

		s.mutex.Lock()
		defer s.mutex.Unlock()

		// TODO: check if still subscribed, otherwise discard

		if err := status.Error(); err != nil {
			log.Warn(s, err.Error(), "object", name)
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
		s.consumeCheck(node, hash)
	})
}
