package sync

import enc "github.com/named-data/ndnd/std/encoding"

type Snapshot interface {
	// Snapshot returns the Snapshot trait.
	Snapshot() Snapshot

	// setName sets the name of the node and group.
	setNames(node enc.Name, group enc.Name)

	// setCallback sets the callback for fetched snapshot.
	//
	// The callback provides a function to update the state vector,
	// and return the snapshot publication. When updating the state vector,
	// make sure to only update the following fields. Updating pending is
	// required, otherwise the fetcher will break.
	//
	//   - SnapBlock - to unblock fetching for the node
	//   - Known - set to max(Known, SnapSeq)
	//   - Pending - set to max(Pending, Known)
	//
	// The name of the snapshot publication must either be a node name
	// when a single node is affected, or empty to indicate the entire group
	// has been updated (i.e. one or more nodes).
	//
	// Only Publisher, Content and DataName fields in the pub are required.
	// Other fields are informational and the application can ignore them.
	//
	setCallback(snapshotCallback)

	// check is called when the state vector is updated.
	// The strategy can decide to block fetching for the snapshot.
	//
	// This function call MUST NOT make the callback.
	check(snapshotOnUpdateArgs)
}

type snapshotOnUpdateArgs struct {
	// state is the current state vector.
	state SvMap[svsDataState]
	// node is the node that is updated.
	node enc.Name
	// hash is the hash of the node name (optimization).
	hash string
}

type snapshotCallbackInner = func(state SvMap[svsDataState]) (SvsPub, bool)
type snapshotCallback = func(callback snapshotCallbackInner)
