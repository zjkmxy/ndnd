package utils_test

import (
	"testing"
	"time"

	"github.com/named-data/ndnd/std/utils"
	tu "github.com/named-data/ndnd/std/utils/testutils"
	"github.com/stretchr/testify/require"
)

func TestIdPtr(t *testing.T) {
	tu.SetT(t)

	p := utils.IdPtr(uint64(42))
	require.Equal(t, uint64(42), *p)
}

func TestMakeTimestamp(t *testing.T) {
	tu.SetT(t)

	date := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	require.Equal(t, uint64(1609459200000), utils.MakeTimestamp(date))
}

func TestConvertNonce(t *testing.T) {
	tu.SetT(t)

	nonce := []byte{0x01, 0x02, 0x03, 0x04}
	val := utils.ConvertNonce(nonce)
	require.Equal(t, uint32(0x01020304), val.Unwrap())

	nonce = []byte{0x42, 0x1C, 0xE1, 0x4B}
	val = utils.ConvertNonce(nonce)
	require.Equal(t, uint32(0x421ce14b), val.Unwrap())
}
