//go:build js && wasm

package js

import "syscall/js"

func SliceToJsArray(slice []byte) js.Value {
	jsSlice := js.Global().Get("Uint8Array").New(len(slice))
	js.CopyBytesToJS(jsSlice, slice)
	return jsSlice
}

func JsArrayToSlice(jsSlice js.Value) []byte {
	slice := make([]byte, jsSlice.Get("length").Int())
	js.CopyBytesToGo(slice, jsSlice)
	return slice
}
