package security

import (
	"github.com/named-data/ndnd/std/security/crypto"

	"github.com/named-data/ndnd/std/ndn"
)

func NewNullSigner() ndn.Signer {
	return crypto.NewNullSigner()
}

func NewSha256Signer() ndn.Signer {
	return crypto.NewSha256Signer()
}

func NewHmacSigner(key []byte) ndn.Signer {
	return crypto.NewHmacSigner(key)
}
