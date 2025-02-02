package sync

import (
	"fmt"

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
	// The state should encode the entire state of the node,
	// and should replace any previous publications completely.
	//
	// If this snapshot is delivered to a node, previous publications will
	// be ignored by the receiving node.
	SnapMe func() (enc.Wire, error)
	// Threshold is the number of updates before a snapshot is taken.
	Threshold uint64

	// nodePrefix is the name of this instance.
	nodePrefix enc.Name
	// groupPrefix is the groupPrefix name.
	groupPrefix enc.Name
	// callback is the snapshot callback.
	callback snapshotCallbackWrap

	// prevSeq is my last snapshot sequence number.
	prevSeq uint64
}

func (s *SnapshotNodeLatest) Snapshot() Snapshot {
	return s
}

func (s *SnapshotNodeLatest) initialize(node enc.Name, group enc.Name) {
	if s.Client == nil || s.SnapMe == nil || s.Threshold == 0 {
		panic("SnapshotNodeLatest: not initialized")
	}

	s.nodePrefix = node
	s.groupPrefix = group
}

func (s *SnapshotNodeLatest) setCallback(callback snapshotCallbackWrap) {
	s.callback = callback
}

// check determines if a snapshot should be taken or fetched.
func (s *SnapshotNodeLatest) check(args snapshotOnUpdateArgs) {
	// We only care about the latest boot.
	// For all other states, make sure the fetch is skipped.
	entries := args.state[args.hash]
	for i := range entries {
		if i == len(entries)-1 { // if is last entry
			last := entries[i]  // note: copy
			lastV := last.Value // note: copy

			if args.node.Equal(s.nodePrefix) {
				// This is me - check if I should snapshot
				// 1. I have reached the threshold
				// 2. I have not taken any snapshot yet
				if lastV.Latest-s.prevSeq >= s.Threshold || (s.prevSeq == 0 && lastV.Latest > 0) {
					s.snap(last.Boot, lastV.Latest)
				}
			} else {
				// This is not me - check if I should fetch
				// 1. Pending gap is more than 2*threshold
				// 2. I have not fetched anything yet
				// And, I'm not already blocked by a fetch
				if lastV.SnapBlock == 0 && (lastV.Latest-lastV.Pending >= s.Threshold*2 || lastV.Pending == 0) {
					entries[i].Value.SnapBlock = 1 // released by fetch callback
					s.fetch(args.node, entries[i].Boot)
				}
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

// snapName is the naming convention for snapshots.
func (s *SnapshotNodeLatest) snapName(node enc.Name, boot uint64) enc.Name {
	return node.
		Append(s.groupPrefix...).
		Append(enc.NewTimestampComponent(boot)).
		Append(enc.NewKeywordComponent("snap"))
}

// fetch fetches the latest snapshot.
func (s *SnapshotNodeLatest) fetch(node enc.Name, boot uint64) {
	// Discover the latest snapshot
	s.Client.Consume(s.snapName(node, boot), func(cstate ndn.ConsumeState) {
		if !cstate.IsComplete() {
			return
		}

		s.callback(func(state SvMap[svsDataState]) (pub SvsPub, err error) {
			hash := node.String()
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
	})
}

// snap takes a snapshot of the application state.
func (s *SnapshotNodeLatest) snap(boot uint64, seq uint64) {
	wire, err := s.SnapMe()
	if err != nil {
		log.Error(nil, "Failed to get snapshot", "err", err)
		return
	}

	name := s.snapName(s.nodePrefix, boot).WithVersion(seq)
	name, err = s.Client.Produce(ndn.ProduceArgs{
		Name:    name,
		Content: wire,
	})
	if err != nil {
		log.Error(nil, "Failed to publish snapshot", "err", err, "name", name)
		return
	}

	// TODO: FIFO directory
	s.prevSeq = seq
}
