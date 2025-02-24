//go:build js && wasm

package object

import (
	"syscall/js"

	"unsafe"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/types/priority_queue"
	jsutil "github.com/named-data/ndnd/std/utils/js"
)

type JsStore struct {
	api js.Value

	cache     map[string]*priority_queue.Item[jsStoreTuple, int]
	pq        priority_queue.Queue[jsStoreTuple, int]
	cacheSize int
	cacheP    int
}

type jsStoreTuple struct {
	name string
	wire []byte
}

func NewJsStore(api js.Value) *JsStore {
	return &JsStore{
		api: api,

		cache:     make(map[string]*priority_queue.Item[jsStoreTuple, int], 8192),
		pq:        priority_queue.New[jsStoreTuple, int](),
		cacheSize: 8192, // approx 64MB
		cacheP:    0,
	}
}

func (s *JsStore) Get(name enc.Name, prefix bool) ([]byte, error) {
	s.cacheP++ // priority

	// JS is single-threaded, so no need to lock
	nameTlvStr := name.TlvStr()
	if item, ok := s.cache[nameTlvStr]; ok {
		s.pq.UpdatePriority(item, s.cacheP)
		return item.Value().wire, nil
	}

	name_js := jsutil.SliceToJsArray(name.BytesInner())
	prefix_js := js.ValueOf(prefix)

	// Preload from the store - hint for the last item in page
	var last_hint_js js.Value = js.Undefined()
	if seg := name.At(-1); !prefix && seg.Typ == enc.TypeSegmentNameComponent {
		lastHint := name.Prefix(-1).
			Append(enc.NewSegmentComponent(seg.NumberVal() + 63)). // inclusive
			BytesInner()
		last_hint_js = jsutil.SliceToJsArray(lastHint)
	}

	// [Uint8Array, Uint8Array][]
	page, err := jsutil.Await(s.api.Call("get", name_js, prefix_js, last_hint_js))
	if err != nil {
		return nil, err
	}
	pageSize := page.Get("length").Int()

	// Preload all items in the page
	for i := range pageSize {
		item := page.Index(i)
		name := jsutil.JsArrayToSlice(item.Index(0))
		wire := jsutil.JsArrayToSlice(item.Index(1))

		tlvstr := unsafe.String(unsafe.SliceData(name), len(name)) // no copy
		s.insertCache(tlvstr, wire)

		// If prefix is set, exactly one item should be returned
		if prefix {
			return wire, nil
		}
	}

	if item, ok := s.cache[nameTlvStr]; ok {
		return item.Value().wire, nil
	}

	return nil, nil
}

func (s *JsStore) Put(name enc.Name, version uint64, wire []byte) error {
	tlvBytes := name.BytesInner()
	name_js := jsutil.SliceToJsArray(tlvBytes)
	version_js := js.ValueOf(version)
	wire_js := jsutil.SliceToJsArray(wire)

	// This cannot be awaited because it will block the main thread
	// and deadlock if called from a js function
	s.api.Call("put", name_js, version_js, wire_js) // yolo

	// Cache the item
	tlvStr := unsafe.String(unsafe.SliceData(tlvBytes), len(tlvBytes)) // no copy
	s.insertCache(tlvStr, wire)

	return nil
}

func (s *JsStore) Remove(name enc.Name, prefix bool) error {
	name_js := jsutil.SliceToJsArray(name.BytesInner())
	prefix_js := js.ValueOf(prefix)

	// This does not evict the cache, but that's fine.
	// Applications should not rely on the cache for correctness.

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

func (s *JsStore) insertCache(tlvstr string, wire []byte) {
	if s.cache[tlvstr] == nil {
		s.cache[tlvstr] = s.pq.Push(jsStoreTuple{
			name: tlvstr,
			wire: wire,
		}, s.cacheP)

		// Evict the least recently used item
		if s.pq.Len() > s.cacheSize {
			delete(s.cache, s.pq.Pop().name)
		}
	}
}
