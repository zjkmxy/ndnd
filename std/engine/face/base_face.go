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
	onUp    func()
	onDown  func()
	sendMut sync.Mutex
}

func newBaseFace(local bool) baseFace {
	return baseFace{
		local:  local,
		onUp:   func() {},
		onDown: func() {},
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

func (f *baseFace) OnUp(onUp func()) {
	f.onUp = onUp
}

func (f *baseFace) OnDown(onDown func()) {
	f.onDown = onDown
}

// setStateDown sets the face to down state, and makes the down
// callback if the face was previously up.
func (f *baseFace) setStateDown() {
	if f.running.Swap(false) {
		if f.onDown != nil {
			f.onDown()
		}
	}
}

// setStateUp sets the face to up state, and makes the up
// callback if the face was previously down.
func (f *baseFace) setStateUp() {
	if !f.running.Swap(true) {
		if f.onUp != nil {
			f.onUp()
		}
	}
}

// setStateClosed sets the face to closed state without
// making the onDown callback. Returns if the face was running.
func (f *baseFace) setStateClosed() bool {
	return f.running.Swap(false)
}
