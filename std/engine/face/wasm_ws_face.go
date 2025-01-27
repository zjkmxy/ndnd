//go:build js && wasm

package face

import (
	"errors"
	"syscall/js"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
)

type WasmWsFace struct {
	baseFace
	url  string
	conn js.Value
}

func (f *WasmWsFace) String() string {
	return "wasm-ws-face"
}

func (f *WasmWsFace) Trait() Face {
	return f
}

func NewWasmWsFace(url string, local bool) *WasmWsFace {
	return &WasmWsFace{
		baseFace: newBaseFace(local),
		url:      url,
		conn:     js.Null(),
	}
}

func (f *WasmWsFace) Open() error {
	if f.IsRunning() {
		return errors.New("face is already running")
	}

	if f.onError == nil || f.onPkt == nil {
		return errors.New("face callbacks are not set")
	}

	// It seems now Go cannot handle exceptions thrown by JS
	conn := js.Global().Get("WebSocket").New(f.url)
	conn.Set("binaryType", "arraybuffer")

	ch := make(chan struct{}, 1)
	conn.Call("addEventListener", "message", js.FuncOf(f.receive))
	conn.Call("addEventListener", "open", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		ch <- struct{}{}
		close(ch)
		return nil
	}))

	log.Info(f, "Waiting for WebSocket connection ...")
	<-ch
	log.Info(f, "WebSocket connected")

	f.conn = conn
	f.setStateUp()

	return nil
}

func (f *WasmWsFace) Close() error {
	if f.setStateClosed() {
		f.conn.Call("close")
		f.conn = js.Null()
	}

	return nil
}

func (f *WasmWsFace) Send(pkt enc.Wire) error {
	if !f.IsRunning() {
		return errors.New("face is not running")
	}

	l := pkt.Length()
	arr := js.Global().Get("Uint8Array").New(int(l))
	js.CopyBytesToJS(arr, pkt.Join())
	f.conn.Call("send", arr)

	return nil
}

func (f *WasmWsFace) receive(this js.Value, args []js.Value) any {
	event := args[0]
	data := event.Get("data")
	if !data.InstanceOf(js.Global().Get("ArrayBuffer")) {
		return nil
	}

	buf := make([]byte, data.Get("byteLength").Int())
	view := js.Global().Get("Uint8Array").New(data)
	js.CopyBytesToGo(buf, view)

	err := f.onPkt(buf)
	if err != nil {
		f.Close()
	}

	return nil
}
