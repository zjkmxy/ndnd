package sync

import (
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
)

type seqFetchState struct {
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
}

type svsPubOut struct {
	pub  SvsPub
	subs []func(SvsPub)
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
	if !s.nodePs.HasSub(node) {
		return
	}

	totalPending := uint64(0)

	for _, entry := range s.state[hash] {
		fstate := entry.Value
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
			s.state.Set(hash, entry.Boot, fstate)
			s.consumeObject(node, entry.Boot, fstate.Pending)
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

		if err := status.Error(); err != nil {
			if !s.nodePs.HasSub(node) {
				return // no longer subscribed
			}

			log.Warn(s, err.Error(), "object", name) // TODO: remove

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
			entry.Known = nextSeq
		}

		// Deliver the data with unlocked mutex
		if len(deliver) > 0 {
			// We got these subs when trying to deliver. It doesn't matter
			// if they were unsubscribed after this function exits. The application
			// needs to handle the callbacks correctly.
			subs := s.nodePs.Subs(node)
			if len(subs) == 0 {
				return // no longer subscribed
			}

			// we WILL deliver it, update known state
			s.state.Set(hash, boot, entry)
			for _, pub := range deliver {
				s.outpipe <- svsPubOut{pub, subs}
			}
		}

		// Check if we can fetch more
		s.consumeCheck(node, hash)
	})
}
