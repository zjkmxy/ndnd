package crypto

import (
	"crypto/sha256"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

// Sha256Signer is a Data signer that uses DigestSha256.
type Sha256Signer struct{}

func (Sha256Signer) Type() ndn.SigType {
	return ndn.SignatureDigestSha256
}

func (Sha256Signer) KeyLocator() enc.Name {
	return nil
}

func (Sha256Signer) EstimateSize() uint {
	return 32
}

func (Sha256Signer) Sign(covered enc.Wire) ([]byte, error) {
	h := sha256.New()
	for _, buf := range covered {
		_, err := h.Write(buf)
		if err != nil {
			return nil, enc.ErrUnexpected{Err: err}
		}
	}
	return h.Sum(nil), nil
}

func (Sha256Signer) Public() ([]byte, error) {
	return nil, ndn.ErrNoPubKey
}

// NewSha256Signer creates a signer that uses DigestSha256.
func NewSha256Signer() ndn.Signer {
	return Sha256Signer{}
}
