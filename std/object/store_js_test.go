//go:build js && wasm

package object_test

import (
	"syscall/js"
	"testing"

	"github.com/named-data/ndnd/std/object"
	tu "github.com/named-data/ndnd/std/utils/testutils"
)

func TestJsStore(t *testing.T) {
	tu.SetT(t)
	store := object.NewJsStore(js.Global().Get("_ndnd_store_js"))
	testStoreBasic(t, store)
	// testStoreTxn(t, store)
}
