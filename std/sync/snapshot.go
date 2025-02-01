package sync

import enc "github.com/named-data/ndnd/std/encoding"

type Snapshot interface {
	// Snapshot returns the snapshot trait.
	Snapshot() Snapshot

	// OnUpdate is called when the state vector is updated.
	// The strategy can decide to block fetching for the snapshot.
	// Any fetching in the pipeline will continue.
	//
	// This function call MUST NOT make the callback.
	OnUpdate(args SnapshotOnUpdateArgs)

	// SetCallback sets the callback for fetched snapshot.
	// The callback should provide the snapshot data and
	// the updated state vector with affected nodes.
	// All affected nodes will be unblocked.
	SetCallback(enc.Name)
}

type SnapshotOnUpdateArgs struct {
	// State is the current state vector.
	State SvMap[SvsDataState]
	// Node is the node that is updated.
	Node enc.Name
	// NodeHash is the hash of the node.
	NodeHash string
	// Boot is the updated boot time.
	Boot uint64
	// Updated is the updated state.
	Updated SvsDataState
}
