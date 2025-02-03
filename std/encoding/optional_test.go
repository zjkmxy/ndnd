package encoding_test

import (
	"testing"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/stretchr/testify/require"
)

func TestOptional(t *testing.T) {
	opt := enc.Some[int](42)
	require.True(t, opt.IsSet())
	val, ok := opt.Get()
	require.Equal(t, 42, val)
	require.True(t, ok)
	require.Equal(t, 42, opt.Unwrap())
	require.Equal(t, 42, opt.GetOr(5))

	opt = enc.None[int]()
	require.False(t, opt.IsSet())
	val, ok = opt.Get()
	require.Equal(t, 0, val)
	require.False(t, ok)
	require.Panics(t, func() { opt.Unwrap() })
	require.Equal(t, 5, opt.GetOr(5))

	opt.Set(45)
	require.True(t, opt.IsSet())
	val, ok = opt.Get()
	require.Equal(t, 45, val)
	require.True(t, ok)
	require.Equal(t, 45, opt.Unwrap())
	require.Equal(t, 45, opt.GetOr(5))
}
