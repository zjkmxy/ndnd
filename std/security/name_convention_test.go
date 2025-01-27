package security_test

import (
	"testing"

	enc "github.com/named-data/ndnd/std/encoding"
	sec "github.com/named-data/ndnd/std/security"
	tu "github.com/named-data/ndnd/std/utils/testutils"
	"github.com/stretchr/testify/require"
)

func TestKeyName(t *testing.T) {
	tu.SetT(t)

	id, _ := enc.NameFromStr("/my/test/identity")
	keyPfx, _ := enc.NameFromStr("/my/test/identity/KEY")

	keyName := sec.MakeKeyName(id)
	require.True(t, keyPfx.IsPrefix(keyName))
	require.Equal(t, len(keyPfx)+1, len(keyName))

	id2, _ := sec.GetIdentityFromKeyName(keyName)
	require.Equal(t, id, id2)
}
