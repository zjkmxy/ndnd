package utils

import (
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"golang.org/x/exp/constraints"
)

// NDNd version from source control
// Note: this is only defined in NDNd itself, not for other projects.
var NDNdVersion string = "unknown"

// IdPtr is the pointer version of id: 'a->'a
func IdPtr[T any](value T) *T {
	return &value
}

// ConvIntPtr converts an integer pointer to another type
func ConvIntPtr[A, B constraints.Integer](a *A) *B {
	if a == nil {
		return nil
	} else {
		b := B(*a)
		return &b
	}
}

// ConvOptional converts an optional value to another type
func ConvIntOpt[A, B constraints.Integer](a enc.Optional[A]) (out enc.Optional[B]) {
	if a.IsSet() {
		out.Set(B(a.Unwrap()))
	}
	return out
}

func MakeTimestamp(t time.Time) uint64 {
	return uint64(t.UnixNano() / int64(time.Millisecond))
}

func ConvertNonce(nonce []byte) (ret enc.Optional[uint32]) {
	x := uint32(0)
	for _, b := range nonce {
		x = (x << 8) | uint32(b)
	}
	ret.Set(x)
	return ret
}
