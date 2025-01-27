package face

import (
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"

	enc "github.com/named-data/ndnd/std/encoding"
	ndn_io "github.com/named-data/ndnd/std/utils/io"
)

type StreamFace struct {
	network string
	addr    string
	local   bool
	conn    net.Conn
	running atomic.Bool
	onPkt   func(frame []byte) error
	onError func(err error) error
	sendMut sync.Mutex
}

func (f *StreamFace) String() string {
	return "stream-face"
}

func (f *StreamFace) Trait() Face {
	return f
}

func (f *StreamFace) Run() {
	err := ndn_io.ReadTlvStream(f.conn, func(b []byte) bool {
		if err := f.onPkt(b); err != nil {
			return false // break
		}
		return f.IsRunning()
	}, nil)
	if err != nil {
		f.onError(err)
	} else {
		f.onError(io.EOF)
	}

	f.running.Store(false)
	f.conn = nil
}

func (f *StreamFace) Open() error {
	if f.onError == nil || f.onPkt == nil {
		return errors.New("face callbacks are not set")
	}
	if f.conn != nil {
		return errors.New("face is already running")
	}
	c, err := net.Dial(f.network, f.addr)
	if err != nil {
		return err
	}
	f.conn = c
	f.running.Store(true)
	go f.Run()
	return nil
}

func (f *StreamFace) Close() error {
	if f.conn == nil {
		return errors.New("face is not running")
	}
	f.running.Store(false)
	err := f.conn.Close()
	// f.conn = nil // No need to do so, as Run() will set conn = nil
	return err
}

func (f *StreamFace) Send(pkt enc.Wire) error {
	if !f.running.Load() {
		return errors.New("face is not running")
	}
	f.sendMut.Lock()
	defer f.sendMut.Unlock()
	_, err := f.conn.Write(pkt.Join())
	if err != nil {
		return err
	}
	return nil
}

func (f *StreamFace) IsRunning() bool {
	return f.running.Load()
}

func (f *StreamFace) IsLocal() bool {
	return f.local
}

func (f *StreamFace) SetCallback(onPkt func(frame []byte) error,
	onError func(err error) error) {
	f.onPkt = onPkt
	f.onError = onError
}

func NewStreamFace(network string, addr string, local bool) *StreamFace {
	return &StreamFace{
		network: network,
		addr:    addr,
		local:   local,
		onPkt:   nil,
		onError: nil,
		conn:    nil,
		running: atomic.Bool{},
	}
}
