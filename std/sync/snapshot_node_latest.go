package sync

import (
	"fmt"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
)

// SnapshotNodeLatest is a snapshot strategy that takes a snapshot of the
// application state whenever a certain number of updates have been made.
//
// Each snapshot is treated as self-contained and replaces any previous
// publications completely. Only the latest (hence the name) snapshot is
// fetched by other nodes, and previous publications are ignored.
//
// When a node bootstraps again, this strategy assumes that the previous
// state is now invalid and fetches the latest snapshot.
type SnapshotNodeLatest struct {
	// Client is the object client.
	Client ndn.Client

	// SnapMe is the callback to get a snapshot of the application state.
	//
	// The state should encode the entire state of the node, and should replace
	// any previous publications completely. If this snapshot is delivered to a
	// node, previous publications will be ignored by the receiving node.
	//
	// The callback is passed the name of the snapshot that will be created.
	// The application may insert this name in a FIFO directory to manage storage
	// and remove old publications and snapshots.
	SnapMe func(enc.Name) (enc.Wire, error)
	// Threshold is the number of updates before a snapshot is taken.
	Threshold uint64

	// pss is the struct from the svs layer.
	pss snapPsState
	// prevSeq is my last snapshot sequence number.
	prevSeq uint64
}

func (s *SnapshotNodeLatest) Snapshot() Snapshot {
	return s
}

func (s *SnapshotNodeLatest) initialize(comm snapPsState) {
	if s.Client == nil || s.SnapMe == nil || s.Threshold == 0 {
		panic("SnapshotNodeLatest: not initialized")
	}

	s.pss = comm
}

// checkFetch determines if a snapshot should be fetched.
func (s *SnapshotNodeLatest) checkFetch(args snapCheckArgs) {
	// We only care about the latest boot.
	// For all other states, make sure the fetch is skipped.
	entries := args.state[args.hash]
	for i := range entries {
		if i == len(entries)-1 { // if is last entry
			boot, value := entries[i].Boot, entries[i].Value

			// Check if we should fetch a snapshot
			// 1. Pending gap is more than 2*threshold
			// 2. I have not fetched anything yet
			// And, I'm not already blocked by a fetch
			if value.SnapBlock == 0 && (value.Latest-value.Pending >= s.Threshold*2 || value.Pending == 0) {
				entries[i].Value.SnapBlock = 1 // released by fetch callback
				s.fetch(args.node, boot)
			}
			return
		}

		// Block all older boots permanently
		if entries[i].Value.SnapBlock != -1 {
			entries[i].Value.SnapBlock = -1
			entries[i].Value.Known = entries[i].Value.Latest
			entries[i].Value.Pending = entries[i].Value.Latest
		}
	}
}

// checkSelf is called when the state for this node is updated.
func (s *SnapshotNodeLatest) checkSelf(delivered SvMap[uint64]) {
	// This strategy only cares about the latest boot.
	boots := delivered[s.pss.nodePrefix.TlvStr()]
	entry := boots[len(boots)-1]

	// Check if I should take a snapshot
	// 1. I have reached the threshold
	// 2. I have not taken any snapshot yet
	if entry.Value-s.prevSeq >= s.Threshold || (s.prevSeq == 0 && entry.Value > 0) {
		s.prevSeq = entry.Value
		s.takeSnap(entry.Boot, entry.Value)
	}
}

// snapName is the naming convention for snapshots.
func (s *SnapshotNodeLatest) snapName(node enc.Name, boot uint64) enc.Name {
	return node.
		Append(s.pss.groupPrefix...).
		Append(enc.NewTimestampComponent(boot)).
		Append(enc.NewKeywordComponent("SNAP"))
}

// fetch fetches the latest snapshot.
func (s *SnapshotNodeLatest) fetch(node enc.Name, boot uint64) {
	// Discover the latest snapshot
	s.Client.Consume(s.snapName(node, boot), func(cstate ndn.ConsumeState) {
		if cstate.Error() != nil {
			// Do not try too fast in case NFD returns NACK
			time.AfterFunc(2*time.Second, func() {
				s.handleSnap(node, boot, cstate)
			})
		} else {
			s.handleSnap(node, boot, cstate)
		}
	})
}

func (s *SnapshotNodeLatest) handleSnap(node enc.Name, boot uint64, cstate ndn.ConsumeState) {
	s.pss.onReceive(func(state SvMap[svsDataState]) (pub SvsPub, err error) {
		hash := node.TlvStr()
		pub.Publisher = node

		if err := cstate.Error(); err != nil {
			// Unblock the state - will lead back to us
			entry := state.Get(hash, boot)
			if entry.SnapBlock == 1 {
				entry.SnapBlock = 0
				state.Set(hash, boot, entry)
			}
			return pub, err
		}

		entry := state.Get(hash, boot)
		if entry.SnapBlock != 1 || entry.Known >= cstate.Version() {
			return pub, fmt.Errorf("fetched invalid snapshot")
		}

		entry.SnapBlock = 0
		entry.Known = cstate.Version()
		entry.Pending = max(entry.Pending, entry.Known)
		state.Set(hash, boot, entry)

		return SvsPub{
			Publisher:  node,
			Content:    cstate.Content(),
			DataName:   cstate.Name(),
			BootTime:   boot,
			SeqNum:     cstate.Version(),
			IsSnapshot: true,
		}, nil
	})
}

// takeSnap takes a snapshot of the application state.
func (s *SnapshotNodeLatest) takeSnap(boot uint64, seq uint64) {
	name := s.snapName(s.pss.nodePrefix, boot).WithVersion(seq)

	// Request snapshot from application
	wire, err := s.SnapMe(name)
	if err != nil {
		log.Error(nil, "Failed to get snapshot", "err", err)
		return
	}

	// Publish snapshot into our store
	name, err = s.Client.Produce(ndn.ProduceArgs{
		Name:    name,
		Content: wire,
	})
	if err != nil {
		log.Error(nil, "Failed to publish snapshot", "err", err, "name", name)
		return
	}
}
