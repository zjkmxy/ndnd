//go:build js && wasm

package utils

import (
	"errors"
	"syscall/js"
)

func AsyncFunc(f func(this js.Value, p []js.Value) (any, error)) js.Func {
	return js.FuncOf(func(this js.Value, p []js.Value) interface{} {
		promise, resolve, reject := Promise()
		go func() {
			ret, err := f(this, p)
			if err != nil {
				reject(err.Error())
			} else {
				resolve(ret)
			}
		}()
		return promise
	})
}

func Promise() (promise js.Value, resolve func(args ...any), reject func(args ...any)) {
	var jsResolve, jsReject js.Value
	return js.Global().Get("Promise").New(js.FuncOf(func(this js.Value, args []js.Value) any {
			jsResolve = args[0]
			jsReject = args[1]
			return nil
		})), func(args ...any) {
			jsResolve.Invoke(args...)
		}, func(args ...any) {
			jsReject.Invoke(args...)
		}
}

func Await(promise js.Value) (val js.Value, err error) {
	resc := make(chan js.Value, 1)
	errc := make(chan error, 1)
	promise.Call("then", js.FuncOf(func(this js.Value, p []js.Value) interface{} {
		resc <- p[0]
		return nil
	})).Call("catch", js.FuncOf(func(this js.Value, p []js.Value) interface{} {
		errc <- errors.New(p[0].String())
		return nil
	}))
	select {
	case r := <-resc:
		return r, nil
	case e := <-errc:
		return js.Undefined(), e
	}
}
