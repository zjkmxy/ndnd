//go:build js && wasm

package face

import (
	"errors"
	"sync/atomic"
	"syscall/js"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
)

type WasmWsFace struct {
	url     string
	local   bool
	conn    js.Value
	running atomic.Bool
	onPkt   func(frame []byte) error
	onError func(err error) error
}

func (f *WasmWsFace) String() string {
	return "wasm-sim-face"
}

func (f *WasmWsFace) Trait() Face {
	return f
}

func (f *WasmWsFace) onMessage(this js.Value, args []js.Value) any {
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
		f.running.Store(false)
		f.conn.Call("close")
		f.conn = js.Null()
	}
	return nil
}

func (f *WasmWsFace) Send(pkt enc.Wire) error {
	if !f.running.Load() {
		return errors.New("face is not running")
	}
	l := pkt.Length()
	arr := js.Global().Get("Uint8Array").New(int(l))
	js.CopyBytesToJS(arr, pkt.Join())
	f.conn.Call("send", arr)
	return nil
}

func (f *WasmWsFace) Open() error {
	if f.onError == nil || f.onPkt == nil {
		return errors.New("face callbacks are not set")
	}
	if !f.conn.IsNull() {
		return errors.New("face is already running")
	}
	ch := make(chan struct{}, 1)
	// It seems now Go cannot handle exceptions thrown by JS
	f.conn = js.Global().Get("WebSocket").New(f.url)
	f.conn.Set("binaryType", "arraybuffer")
	f.conn.Call("addEventListener", "open", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		ch <- struct{}{}
		close(ch)
		return nil
	}))
	f.conn.Call("addEventListener", "message", js.FuncOf(f.onMessage))
	log.Info(f, "Waiting for WebSocket connection ...")
	<-ch
	log.Info(f, "WebSocket connected")
	f.running.Store(true)
	return nil
}

func (f *WasmWsFace) Close() error {
	if f.conn.IsNull() {
		return errors.New("face is not running")
	}
	f.running.Store(false)
	f.conn.Call("close")
	f.conn = js.Null()
	return nil
}

func (f *WasmWsFace) IsRunning() bool {
	return f.running.Load()
}

func (f *WasmWsFace) IsLocal() bool {
	return f.local
}

func (f *WasmWsFace) SetCallback(onPkt func(frame []byte) error,
	onError func(err error) error) {
	f.onPkt = onPkt
	f.onError = onError
}

func NewWasmWsFace(url string, local bool) *WasmWsFace {
	return &WasmWsFace{
		url:     url,
		local:   local,
		onPkt:   nil,
		onError: nil,
		conn:    js.Null(),
		running: atomic.Bool{},
	}
}
