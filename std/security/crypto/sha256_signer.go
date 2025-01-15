package crypto

import (
	"crypto/sha256"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

// sha256Signer is a Data signer that uses DigestSha256.
type sha256Signer struct{}

func (sha256Signer) Type() ndn.SigType {
	return ndn.SignatureDigestSha256
}

func (sha256Signer) EstimateSize() uint {
	return 32
}

func (sha256Signer) Sign(covered enc.Wire) ([]byte, error) {
	h := sha256.New()
	for _, buf := range covered {
		_, err := h.Write(buf)
		if err != nil {
			return nil, enc.ErrUnexpected{Err: err}
		}
	}
	return h.Sum(nil), nil
}

// NewSha256Signer creates a signer that uses DigestSha256.
func NewSha256Signer() ndn.CryptoSigner {
	return sha256Signer{}
}
