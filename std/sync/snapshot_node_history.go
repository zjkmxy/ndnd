package sync

import (
	"fmt"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/ndn/svs_ps"
)

// SnapshotNodeLatest is a snapshot strategy that assumes that it is not
// possible to take a snapshot of the application state. Instead, it creates
// a snapshot of the entire publication history.
//
// This strategy should be used with highly persistent storage, as it will
// store all publications since the node bootstrapped, and fetch publications
// from a node's previous instances (bootstraps). To ensure that publications
// from previous instances are available, the application must use NDN Repo.
type SnapshotNodeHistory struct {
	// Client is the object client.
	Client ndn.Client
	// Threshold is the number of updates before a snapshot is taken.
	Threshold uint64

	// pss is the struct from the svs layer.
	pss snapPsState
	// prevSeq is my last snapshot sequence number.
	prevSeq uint64
}

func (s *SnapshotNodeHistory) String() string {
	return "snapshot-node-history"
}

func (s *SnapshotNodeHistory) Snapshot() Snapshot {
	return s
}

// initialize the snapshot strategy.
func (s *SnapshotNodeHistory) initialize(pss snapPsState) {
	if s.Client == nil || s.Threshold == 0 {
		panic("SnapshotNodeLatest: not initialized")
	}
	s.pss = pss
}

// onUpdate determines if a snapshot should be fetched.
func (s *SnapshotNodeHistory) onUpdate(state SvMap[svsDataState], node enc.Name) {
	entries := state[node.TlvStr()]

	for i := range entries {
		boot, value := entries[i].Boot, entries[i].Value

		// Skip this instance, what's the point?
		if node.Equal(s.pss.nodePrefix) && boot == s.pss.bootTime {
			continue
		}

		// Check if we should fetch a snapshot
		//
		// TODO: prevent fetching snapshot too fast - throttle this call
		// This will prevent an infinite loop if the snapshot is old (?)
		if value.SnapBlock == 0 && (value.Latest-value.Pending >= s.Threshold*2 || value.Known == 0) {
			entries[i].Value.SnapBlock = 1 // released by fetch callback
			s.fetchIndex(node, boot, value.Known)
		}
	}
}

// onPublication is called when the state for this node is updated.
func (s *SnapshotNodeHistory) onPublication(state SvMap[svsDataState], pub enc.Name) {
	// Get the sequence number
	entry := state.Get(s.pss.nodePrefix.TlvStr(), s.pss.bootTime)
	seqNo := entry.Known

	// Check if I should take a snapshot
	// 1. I have reached the threshold
	// 2. I have not taken any snapshot yet
	if seqNo-s.prevSeq >= s.Threshold || (s.prevSeq == 0 && seqNo > 0) {
		s.takeSnap(seqNo)
	}
}

// snapName is the naming convention for snapshots.
func (s *SnapshotNodeHistory) snapName(node enc.Name, boot uint64) enc.Name {
	return node.
		Append(s.pss.groupPrefix...).
		Append(enc.NewTimestampComponent(boot)).
		Append(enc.NewKeywordComponent("HIST"))
}

func (s *SnapshotNodeHistory) idxName(node enc.Name, boot uint64) enc.Name {
	return node.
		Append(s.pss.groupPrefix...).
		Append(enc.NewTimestampComponent(boot)).
		Append(enc.NewKeywordComponent("HIDX"))
}

// fetchIndex fetches the latest index for a remote node.
func (s *SnapshotNodeHistory) fetchIndex(node enc.Name, boot uint64, known uint64) {
	s.Client.Consume(s.idxName(node, boot), func(cstate ndn.ConsumeState) {
		go s.handleIndex(node, boot, known, cstate)
	})
}

// handleIndex processes the fetched index.
func (s *SnapshotNodeHistory) handleIndex(node enc.Name, boot uint64, known uint64, cstate ndn.ConsumeState) {
	hash := node.TlvStr()

	onError := func(err error) {
		time.Sleep(2 * time.Second) // we are in a different goroutine

		s.pss.onSnap(func(state SvMap[svsDataState]) (SvsPub, error) {
			entry := state.Get(hash, boot)
			entry.SnapBlock = 0
			state.Set(hash, boot, entry)

			return SvsPub{}, &ErrSync{
				Publisher: node,
				BootTime:  boot,
				Err:       err,
			}
		})
	}

	// Fetching error. We already debounced in the fetchIndex callback.
	if err := cstate.Error(); err != nil {
		onError(fmt.Errorf("failed to fetch snapshot index: %w", err))
		return
	}

	// Parse the index
	index, err := svs_ps.ParseHistoryIndex(enc.NewWireView(cstate.Content()), true)
	if err != nil {
		onError(fmt.Errorf("failed to parse snapshot index: %w", err))
		return
	}

	// Fetch one snapshot at a time under this goroutine
	for indexI, seqNo := range index.SeqNos {
		if seqNo > known {
			snapC := make(chan ndn.ConsumeState)

			snapName := s.snapName(node, boot).WithVersion(seqNo)
			s.Client.Consume(snapName, func(cstate ndn.ConsumeState) { snapC <- cstate })

			scstate := <-snapC
			if err := scstate.Error(); err != nil {
				onError(fmt.Errorf("failed to fetch snapshot: %w", err))
				return
			}

			// Parse history
			snapshot, err := svs_ps.ParseHistorySnap(enc.NewWireView(scstate.Content()), true)
			if err != nil {
				onError(fmt.Errorf("failed to parse snapshot: %w", err))
				return
			}

			// Verify snapshot has entries till the end
			if len(snapshot.Entries) == 0 || snapshot.Entries[len(snapshot.Entries)-1].SeqNo != scstate.Version() {
				onError(fmt.Errorf("fetched incomplete snapshot - ignoring"))
				return
			}

			s.pss.onSnap(func(state SvMap[svsDataState]) (SvsPub, error) {
				entry := state.Get(hash, boot)

				if entry.Known >= scstate.Version() {
					return SvsPub{}, fmt.Errorf("fetched old snapshot - ignoring")
				}

				// Filter out any publications which are already known
				for len(snapshot.Entries) > 0 {
					if snapshot.Entries[0].SeqNo <= entry.Known {
						snapshot.Entries = snapshot.Entries[1:]
					} else {
						break
					}
				}

				// Update the state vector
				if indexI == len(index.SeqNos)-1 {
					// Reset only if this is the last snapshot in the index
					entry.SnapBlock = 0
				}
				entry.Known = scstate.Version()
				entry.Pending = max(entry.Pending, entry.Known)
				state.Set(hash, boot, entry)

				return SvsPub{
					Publisher:  node,
					Content:    snapshot.Encode(),
					DataName:   scstate.Name(),
					BootTime:   boot,
					SeqNum:     scstate.Version(),
					IsSnapshot: true,
				}, nil
			})
		}
	}
}

// takeSnap takes a snapshot of the application state for the current node.
func (s *SnapshotNodeHistory) takeSnap(seqNo uint64) {
	// Current snapshot index
	prevIndexName, index, err := s.getIndex()
	if err != nil {
		log.Debug(s, "Failed to get previous index, starting fresh", "err", err)
	}
	if index == nil {
		index = &svs_ps.HistoryIndex{}
	}

	// Check version number of previous snapshot
	if len(index.SeqNos) > 0 {
		s.prevSeq = index.SeqNos[len(index.SeqNos)-1]
	}
	if s.prevSeq > 0 && seqNo < s.prevSeq+s.Threshold-1 {
		log.Info(s, "Previous snapshot is still current", "prev", s.prevSeq, "seq", seqNo)
		return
	}

	// Add new sequence number to the index
	index.SeqNos = append(index.SeqNos, seqNo)

	// Create a new snapshot
	snapshot := svs_ps.HistorySnap{
		Entries: make([]*svs_ps.HistorySnapEntry, 0, seqNo-s.prevSeq),
	}

	// Add all publications since the last snapshot
	pubBasename := s.pss.nodePrefix.
		Append(s.pss.groupPrefix...).
		Append(enc.NewTimestampComponent(s.pss.bootTime))
	for i := s.prevSeq + 1; i <= seqNo; i++ {
		pubName := pubBasename.
			Append(enc.NewSequenceNumComponent(i)).
			WithVersion(enc.VersionImmutable)
		content, err := s.Client.GetLocal(pubName)
		if err != nil {
			log.Error(s, "Failed to get publication", "err", err, "name", pubName)
			return
		}
		snapshot.Entries = append(snapshot.Entries, &svs_ps.HistorySnapEntry{
			SeqNo:   i,
			Content: content,
		})
	}

	// Publish snapshot into our store
	snapName := s.snapName(s.pss.nodePrefix, s.pss.bootTime).
		WithVersion(seqNo)
	_, err = s.Client.Produce(ndn.ProduceArgs{
		Name:    snapName,
		Content: snapshot.Encode(),
	})
	if err != nil {
		log.Error(s, "Failed to publish snapshot", "err", err, "name", snapName)
		return
	}

	// Write new index
	indexWire := index.Encode()
	_, err = s.Client.Produce(ndn.ProduceArgs{
		Name: s.idxName(s.pss.nodePrefix, s.pss.bootTime).
			WithVersion(seqNo),
		Content: indexWire,
	})
	if err != nil {
		log.Error(s, "Failed to publish index", "err", err, "name", prevIndexName)
		return
	}

	// Update the sequence number
	s.prevSeq = seqNo

	// Evict old index
	if prevIndexName != nil {
		err = s.Client.Remove(prevIndexName)
		if err != nil {
			log.Warn(s, "Failed to evict old index", "err", err, "name", prevIndexName)
		}
	}
}

func (s *SnapshotNodeHistory) getIndex() (enc.Name, *svs_ps.HistoryIndex, error) {
	idxName := s.idxName(s.pss.nodePrefix, s.pss.bootTime)

	prevIdxName, err := s.Client.LatestLocal(idxName)
	if err != nil {
		return nil, nil, err
	}

	prevIdxWire, err := s.Client.GetLocal(prevIdxName)
	if err != nil {
		return nil, nil, err
	}

	prevIdx, err := svs_ps.ParseHistoryIndex(enc.NewWireView(prevIdxWire), true)
	if err != nil {
		return nil, nil, err
	}

	return prevIdxName, prevIdx, nil
}
