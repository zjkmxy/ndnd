//go:build js && wasm

package object

import (
	"syscall/js"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	jsutil "github.com/named-data/ndnd/std/utils/js"
)

type JsStore struct {
	mem ndn.Store
	api js.Value
}

func NewJsStore(api js.Value) *JsStore {
	return &JsStore{
		mem: NewMemoryStore(),
		api: api,
	}
}

func (s *JsStore) Get(name enc.Name, prefix bool) ([]byte, error) {
	name_js := jsutil.SliceToJsArray(name.BytesInner())
	prefix_js := js.ValueOf(prefix)

	buf, err := jsutil.Await(s.api.Call("get", name_js, prefix_js))
	if err != nil {
		return nil, err
	}

	if buf.IsUndefined() {
		return nil, nil
	}

	return jsutil.JsArrayToSlice(buf), nil
}

func (s *JsStore) Put(name enc.Name, version uint64, wire []byte) error {
	name_js := jsutil.SliceToJsArray(name.BytesInner())
	version_js := js.ValueOf(version)
	wire_js := jsutil.SliceToJsArray(wire)

	// This cannot be awaited because it will block the main thread
	// and deadlock if called from a js function
	s.api.Call("put", name_js, version_js, wire_js) // yolo
	return nil
}

func (s *JsStore) Remove(name enc.Name, prefix bool) error {
	name_js := jsutil.SliceToJsArray(name.BytesInner())
	prefix_js := js.ValueOf(prefix)

	s.api.Call("remove", name_js, prefix_js)
	return nil
}

func (s *JsStore) Begin() (ndn.Store, error) {
	return s, nil
}

func (s *JsStore) Commit() error {
	return nil
}

func (s *JsStore) Rollback() error {
	return nil
}
