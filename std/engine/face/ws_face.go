package face

import (
	"errors"

	"github.com/gorilla/websocket"
	enc "github.com/named-data/ndnd/std/encoding"
)

type WebSocketFace struct {
	baseFace
	url  string
	conn *websocket.Conn
}

func NewWebSocketFace(url string, local bool) *WebSocketFace {
	return &WebSocketFace{
		baseFace: baseFace{
			local: local,
		},
		url: url,
	}
}

func (f *WebSocketFace) Open() error {
	if f.running.Load() {
		return errors.New("face is already running")
	}

	if f.onError == nil || f.onPkt == nil {
		return errors.New("face callbacks are not set")
	}

	c, _, err := websocket.DefaultDialer.Dial(f.url, nil)
	if err != nil {
		return err
	}

	f.conn = c
	f.running.Store(true)

	go f.receive()

	return nil
}

func (f *WebSocketFace) Close() error {
	if !f.running.Swap(false) {
		return errors.New("face is not running")
	}

	return f.conn.Close()
}

func (f *WebSocketFace) Send(pkt enc.Wire) error {
	if !f.running.Load() {
		return errors.New("face is not running")
	}
	return f.conn.WriteMessage(websocket.BinaryMessage, pkt.Join())
}

func (f *WebSocketFace) receive() {
	for f.running.Load() {
		messageType, pkt, err := f.conn.ReadMessage()
		if err != nil {
			if f.running.Load() {
				f.onError(err)
			}
			break
		}

		if messageType != websocket.BinaryMessage {
			continue
		}

		err = f.onPkt(pkt)
		if err != nil {
			break
		}
	}

	f.running.Store(false)
	f.conn = nil
}
