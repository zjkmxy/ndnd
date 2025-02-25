//go:build !js

package object_test

import (
	"os"
	"testing"

	"github.com/named-data/ndnd/std/object"
	tu "github.com/named-data/ndnd/std/utils/testutils"
	"github.com/stretchr/testify/require"
)

func TestBoltStore(t *testing.T) {
	tu.SetT(t)
	filename := "test.db"
	os.Remove(filename)
	defer os.Remove(filename)

	store, err := object.NewBoltStore(filename)
	require.NoError(t, err)
	testStoreBasic(t, store)
	testStoreRemoveRange(t, store)
	testStoreTxn(t, store)
	require.NoError(t, store.Close())
}
