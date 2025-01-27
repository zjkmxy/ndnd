package face

import (
	"sync"
	"sync/atomic"
)

// baseFace is the base struct for face implementations.
type baseFace struct {
	running atomic.Bool
	local   bool
	onPkt   func(frame []byte) error
	onError func(err error) error
	sendMut sync.Mutex
}

func (f *baseFace) IsRunning() bool {
	return f.running.Load()
}

func (f *baseFace) IsLocal() bool {
	return f.local
}

func (f *baseFace) OnPacket(onPkt func(frame []byte) error) {
	f.onPkt = onPkt
}

func (f *baseFace) OnError(onError func(err error) error) {
	f.onError = onError
}
