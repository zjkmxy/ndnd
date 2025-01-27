package face

import (
	"errors"

	enc "github.com/named-data/ndnd/std/encoding"
)

type DummyFace struct {
	baseFace
	sendPkts []enc.Buffer
}

func NewDummyFace() *DummyFace {
	return &DummyFace{
		baseFace: newBaseFace(true),
		sendPkts: make([]enc.Buffer, 0),
	}
}

func (f *DummyFace) Open() error {
	if f.onError == nil || f.onPkt == nil {
		return errors.New("face callbacks are not set")
	}
	if f.running.Load() {
		return errors.New("face is already running")
	}
	f.running.Store(true)
	return nil
}

func (f *DummyFace) Close() error {
	if !f.running.Swap(false) {
		return errors.New("face is not running")
	}
	return nil
}

func (f *DummyFace) Send(pkt enc.Wire) error {
	if !f.running.Load() {
		return errors.New("face is not running")
	}
	if len(pkt) == 1 {
		f.sendPkts = append(f.sendPkts, pkt[0])
	} else if len(pkt) >= 2 {
		newBuf := make(enc.Buffer, 0)
		for _, buf := range pkt {
			newBuf = append(newBuf, buf...)
		}
		f.sendPkts = append(f.sendPkts, newBuf)
	}
	return nil
}

// FeedPacket feeds a packet for the engine to consume
func (f *DummyFace) FeedPacket(pkt enc.Buffer) error {
	if !f.running.Load() {
		return errors.New("face is not running")
	}
	return f.onPkt(pkt)
}

// Consume consumes a packet from the engine
func (f *DummyFace) Consume() (enc.Buffer, error) {
	if !f.running.Load() {
		return nil, errors.New("face is not running")
	}
	if len(f.sendPkts) == 0 {
		return nil, errors.New("no packet to consume")
	}
	pkt := f.sendPkts[0]
	f.sendPkts = f.sendPkts[1:]
	return pkt, nil
}
