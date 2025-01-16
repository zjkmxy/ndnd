package security

import (
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/security/signer"
)

func NewSha256Signer() ndn.Signer {
	return signer.NewSha256Signer()
}

func NewHmacSigner(key []byte) ndn.Signer {
	return signer.NewHmacSigner(key)
}
