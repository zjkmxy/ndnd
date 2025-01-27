package face

import (
	"errors"
	"io"
	"net"

	enc "github.com/named-data/ndnd/std/encoding"
	ndn_io "github.com/named-data/ndnd/std/utils/io"
)

// StreamFace is a face that uses a stream connection.
type StreamFace struct {
	baseFace
	network string
	addr    string
	conn    net.Conn
}

func NewStreamFace(network string, addr string, local bool) *StreamFace {
	return &StreamFace{
		baseFace: baseFace{
			local: local,
		},
		network: network,
		addr:    addr,
	}
}

func (f *StreamFace) String() string {
	return "stream-face"
}

func (f *StreamFace) Trait() Face {
	return f
}

func (f *StreamFace) Open() error {
	if f.running.Load() {
		return errors.New("face is already running")
	}

	if f.onError == nil || f.onPkt == nil {
		return errors.New("face callbacks are not set")
	}

	c, err := net.Dial(f.network, f.addr)
	if err != nil {
		return err
	}

	f.conn = c
	f.running.Store(true)

	go f.receive()

	return nil
}

func (f *StreamFace) Close() error {
	if !f.running.Swap(false) {
		return nil
	}

	if f.conn != nil {
		return f.conn.Close()
	}

	return nil
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

func (f *StreamFace) receive() {
	err := ndn_io.ReadTlvStream(f.conn, func(b []byte) bool {
		if err := f.onPkt(b); err != nil {
			return false // break
		}
		return f.IsRunning()
	}, nil)

	if f.running.Swap(false) {
		if err != nil {
			f.onError(err)
		} else {
			f.onError(io.EOF)
		}
	}
}
