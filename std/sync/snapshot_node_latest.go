package sync

import (
	enc "github.com/named-data/ndnd/std/encoding"
)

const ssthresh = 5 // TODO: configurable

type SnapshotNodeLatest struct {
}

func (s *SnapshotNodeLatest) Snapshot() Snapshot {
	return s
}

func (s *SnapshotNodeLatest) onUpdate(args snapshotOnUpdateArgs) {
	// We only care about the latest boot.
	// For all other states, make sure the fetch is skipped.
	entries := args.state[args.nodeHash]
	for i := range entries {
		if i == len(entries)-1 {
			if args.boot == entries[i].Boot {
				break // latest boot update
			}
			return // old boot update
		}
		// This block is permanent.
		if !entries[i].Value.SnapBlock {
			entries[i].Value.SnapBlock = true
			entries[i].Value.Known = entries[i].Value.Latest
			entries[i].Value.Pending = entries[i].Value.Latest
		}
	}

	if args.entry.Latest-args.entry.Pending > ssthresh {

	}
}

func (s *SnapshotNodeLatest) setCallback(enc.Name) {
}
