package sync

import enc "github.com/named-data/ndnd/std/encoding"

type Snapshot interface {
	// Snapshot returns the Snapshot trait.
	Snapshot() Snapshot

	// initialize the strategy, and set up ps state.
	initialize(snapPsState)

	// checkFetch is called when the state vector is updated for a different node.
	// The strategy can decide to block fetching for the snapshot.
	// This function may also be called for this node name with a different boot time.
	checkFetch(state SvMap[svsDataState], node enc.Name)

	// checkSelf is called when the state for this node is updated (for this boot).
	// The strategy can decide to take a snapshot.
	checkSelf(state SvMap[svsDataState])
}

// snapPsState is the shared data struct between snapshot strategy
// and the SVS data fetching layer.
type snapPsState struct {
	// nodePrefix is the name of the current nodePrefix.
	nodePrefix enc.Name
	// groupPrefix is the name of the sync groupPrefix.
	groupPrefix enc.Name

	// onReceive is the callback for snapshot received from a remote party.
	// The snapshot strategy should call the inner function when
	// a snapshot is received.
	//
	// The callback provides a function to update the state vector,
	// and return the snapshot publication. When updating the state vector,
	// make sure to only update the following fields. Updating Pending is
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
	// Even if the callback returns an error, the Publication field should
	// be appropriately set. This will trigger a re-fetch for the producers.
	//
	onReceive func(callback snapRecvCallback)
}

// snapRecvCallback is the callback function passed to the onReceive callback.
// This callback should update the state if needed (lock is held by the caller).
type snapRecvCallback = func(state SvMap[svsDataState]) (SvsPub, error)
