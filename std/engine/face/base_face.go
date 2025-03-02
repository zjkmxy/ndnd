package face

import (
	"sync"
	"sync/atomic"
)

// baseFace is the base struct for face implementations.
type baseFace struct {
	running atomic.Bool
	local   bool
	onPkt   func(frame []byte)
	onError func(err error)
	sendMut sync.Mutex

	onUp     map[int]func()
	onDown   map[int]func()
	onUpHndl int
	onDnHndl int
}

func newBaseFace(local bool) baseFace {
	return baseFace{
		local:  local,
		onUp:   make(map[int]func()),
		onDown: make(map[int]func()),
	}
}

func (f *baseFace) IsRunning() bool {
	return f.running.Load()
}

func (f *baseFace) IsLocal() bool {
	return f.local
}

func (f *baseFace) OnPacket(onPkt func(frame []byte)) {
	f.onPkt = onPkt
}

func (f *baseFace) OnError(onError func(err error)) {
	f.onError = onError
}

func (f *baseFace) OnUp(onUp func()) (cancel func()) {
	hndl := f.onUpHndl
	f.onUp[hndl] = onUp
	f.onUpHndl++
	return func() {
		delete(f.onUp, hndl)
	}
}

func (f *baseFace) OnDown(onDown func()) (cancel func()) {
	hndl := f.onDnHndl
	f.onDown[hndl] = onDown
	f.onDnHndl++
	return func() {
		delete(f.onDown, hndl)
	}
}

// setStateDown sets the face to down state, and makes the down
// callback if the face was previously up.
func (f *baseFace) setStateDown() {
	if f.running.Swap(false) {
		for _, cb := range f.onDown {
			cb()
		}
	}
}

// setStateUp sets the face to up state, and makes the up
// callback if the face was previously down.
func (f *baseFace) setStateUp() {
	if !f.running.Swap(true) {
		for _, cb := range f.onUp {
			cb()
		}
	}
}

// setStateClosed sets the face to closed state without
// making the onDown callback. Returns if the face was running.
func (f *baseFace) setStateClosed() bool {
	return f.running.Swap(false)
}
