//go:build js && wasm

package face

import (
	"fmt"
	"syscall/js"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	jsutil "github.com/named-data/ndnd/std/utils/js"
)

type WasmWsFace struct {
	baseFace
	url  string
	conn js.Value
}

func NewWasmWsFace(url string, local bool) *WasmWsFace {
	return &WasmWsFace{
		baseFace: newBaseFace(local),
		url:      url,
		conn:     js.Null(),
	}
}

func (f *WasmWsFace) String() string {
	return fmt.Sprintf("wasm-ws-face (%s)", f.url)
}

func (f *WasmWsFace) Trait() Face {
	return f
}

func (f *WasmWsFace) Open() error {
	if f.IsRunning() {
		return fmt.Errorf("face is already running")
	}

	if f.onError == nil || f.onPkt == nil {
		return fmt.Errorf("face callbacks are not set")
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
		return fmt.Errorf("face is not running")
	}

	arr := jsutil.SliceToJsArray(pkt.Join())
	f.conn.Call("send", arr)

	return nil
}

func (f *WasmWsFace) receive(this js.Value, args []js.Value) any {
	event := args[0]
	data := event.Get("data")
	f.onPkt(jsutil.JsArrayToSlice(data))
	return nil
}
