//go:build js && wasm

package face

import (
	"errors"
	"syscall/js"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
)

type WasmSimFace struct {
	baseFace
	gosim js.Value
}

func NewWasmSimFace() *WasmSimFace {
	return &WasmSimFace{
		baseFace: baseFace{
			local: true,
		},
		gosim: js.Null(),
	}
}

func (f *WasmSimFace) String() string {
	return "wasm-sim-face"
}

func (f *WasmSimFace) Trait() Face {
	return f
}

func (f *WasmSimFace) Send(pkt enc.Wire) error {
	if !f.running.Load() {
		return errors.New("face is not running")
	}

	l := pkt.Length()
	arr := js.Global().Get("Uint8Array").New(int(l))
	js.CopyBytesToJS(arr, pkt.Join())
	f.gosim.Call("sendPkt", arr)

	return nil
}

func (f *WasmSimFace) Open() error {
	if f.onError == nil || f.onPkt == nil {
		return errors.New("face callbacks are not set")
	}

	if !f.gosim.IsNull() {
		return errors.New("face is already running")
	}

	// It seems now Go cannot handle exceptions thrown by JS
	f.gosim = js.Global().Get("gondnsim")
	f.gosim.Call("setRecvPktCallback", js.FuncOf(f.receive))

	log.Info(f, "Face opened")
	f.running.Store(true)

	return nil
}

func (f *WasmSimFace) Close() error {
	if f.gosim.IsNull() {
		return errors.New("face is not running")
	}

	f.running.Store(false)
	f.gosim.Call("setRecvPktCallback", js.FuncOf(func(this js.Value, args []js.Value) any {
		return nil
	}))
	f.gosim = js.Null()

	return nil
}

func (f *WasmSimFace) receive(this js.Value, args []js.Value) any {
	pkt := args[0]
	if !pkt.InstanceOf(js.Global().Get("Uint8Array")) {
		return nil
	}

	buf := make([]byte, pkt.Get("byteLength").Int())
	js.CopyBytesToGo(buf, pkt)
	err := f.onPkt(buf)
	if err != nil {
		f.running.Store(false)
		log.Error(f, "Unable to handle packet: %+v", err)
	}

	return nil
}
