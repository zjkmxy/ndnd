//go:build !js

package object_test

import (
	"os"
	"testing"

	"github.com/named-data/ndnd/std/object"
	tu "github.com/named-data/ndnd/std/utils/testutils"
	"github.com/stretchr/testify/require"
)

func TestBadgerStore(t *testing.T) {
	tu.SetT(t)
	dir := "badger-test"
	os.RemoveAll(dir)
	defer os.RemoveAll(dir)

	store, err := object.NewBadgerStore(dir)
	require.NoError(t, err)
	testStoreBasic(t, store)
	testStoreRemoveRange(t, store)
	testStoreTxn(t, store)
	require.NoError(t, store.Close())
}
