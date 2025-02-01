package sync

import enc "github.com/named-data/ndnd/std/encoding"

type Snapshot interface {
	// Snapshot returns the Snapshot trait.
	Snapshot() Snapshot

	// onUpdate is called when the state vector is updated.
	// The strategy can decide to block fetching for the snapshot.
	//
	// This function call MUST NOT make the callback.
	onUpdate(snapshotOnUpdateArgs)

	// setCallback sets the callback for fetched snapshot.
	// The callback should provide the snapshot data and
	// a function to update the state vector.
	setCallback(snapshotCallback)
}

type snapshotOnUpdateArgs struct {
	// state is the current state vector.
	state SvMap[svsDataState]
	// node is the node that is updated.
	node enc.Name
	// nodeHash is the hash of the node.
	nodeHash string
	// boot is the updated boot time.
	boot uint64
	// entry is the updated state.
	entry svsDataState
	// isSelf is true if the update is from self.
	isSelf bool
}

type snapshotUpdater = func(state SvMap[svsDataState])
type snapshotCallback = func(data []byte, updater snapshotUpdater)
