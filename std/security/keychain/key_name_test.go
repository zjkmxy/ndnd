package keychain_test

import (
	"testing"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/security/keychain"
	"github.com/named-data/ndnd/std/utils"
	"github.com/stretchr/testify/require"
)

func TestKeyName(t *testing.T) {
	utils.SetTestingT(t)

	id, _ := enc.NameFromStr("/my/test/identity")
	keyPfx, _ := enc.NameFromStr("/my/test/identity/KEY")

	keyName := keychain.MakeKeyName(id)
	require.True(t, keyPfx.IsPrefix(keyName))
	require.Equal(t, len(keyPfx)+1, len(keyName))

	id2, _ := keychain.GetIdentityFromKeyName(keyName)
	require.Equal(t, id, id2)
}
