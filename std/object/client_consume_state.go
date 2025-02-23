package object

import (
	"sync/atomic"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	rdr "github.com/named-data/ndnd/std/ndn/rdr_2024"
)

// arguments for the consume callback
type ConsumeState struct {
	// original arguments
	args ndn.ConsumeExtArgs
	// error that occurred during fetching
	err error
	// raw data contents.
	content enc.Wire
	// fetching is completed
	complete atomic.Bool
	// fetched metadata
	meta *rdr.MetaData
	// versioned object name
	fetchName enc.Name

	// fetching window
	// - [0] is the position till which the user has already consumed the fetched buffer
	// - [1] is the position till which the buffer is valid (window start)
	// - [2] is the end of the current fetching window
	//
	// content[0:wnd[0]] is invalid (already used and freed)
	// content[wnd[0]:wnd[1]] is valid (not used yet)
	// content[wnd[1]:wnd[2]] is currently being fetched
	// content[wnd[2]:] will be fetched in the future
	wnd [3]int

	// segment count from final block id (-1 if unknown)
	segCnt int
}

// returns the name of the object being consumed
func (a *ConsumeState) Name() enc.Name {
	return a.fetchName
}

// returns the version of the object being consumed
func (a *ConsumeState) Version() uint64 {
	if ver := a.fetchName.At(-1); ver.IsVersion() {
		return ver.NumberVal()
	}
	return 0
}

// returns the error that occurred during fetching
func (a *ConsumeState) Error() error {
	return a.err
}

// returns true if the content has been completely fetched
func (a *ConsumeState) IsComplete() bool {
	return a.complete.Load()
}

// returns the currently available buffer in the content
// any subsequent calls to Content() will return data after the previous call
func (a *ConsumeState) Content() enc.Wire {
	// return valid range of buffer (can be empty)
	wire := make(enc.Wire, a.wnd[1]-a.wnd[0])

	// free buffers
	for i := a.wnd[0]; i < a.wnd[1]; i++ {
		wire[i-a.wnd[0]] = a.content[i] // retain
		a.content[i] = nil              // gc
	}

	a.wnd[0] = a.wnd[1]
	return wire
}

// get the progress counter
func (a *ConsumeState) Progress() int {
	return a.wnd[1]
}

// get the max value for the progress counter (-1 for unknown)
func (a *ConsumeState) ProgressMax() int {
	return a.segCnt
}

// cancel the consume operation
func (a *ConsumeState) Cancel() {
	if !a.complete.Swap(true) {
		a.err = ndn.ErrCancelled
	}
}

// send a fatal error to the callback
func (a *ConsumeState) finalizeError(err error) {
	if !a.complete.Swap(true) {
		a.err = err
		a.args.Callback(a)
	}
}
