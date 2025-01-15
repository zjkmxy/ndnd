package crypto

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

// rsaSigner is a signer that uses ECC key to sign packets.
type rsaSigner struct {
	key *rsa.PrivateKey
}

func (s *rsaSigner) Type() ndn.SigType {
	return ndn.SignatureSha256WithRsa
}

func (s *rsaSigner) EstimateSize() uint {
	return uint(s.key.Size())
}

func (s *rsaSigner) Sign(covered enc.Wire) ([]byte, error) {
	h := sha256.New()
	for _, buf := range covered {
		_, err := h.Write(buf)
		if err != nil {
			return nil, enc.ErrUnexpected{Err: err}
		}
	}
	digest := h.Sum(nil)
	return rsa.SignPKCS1v15(nil, s.key, crypto.SHA256, digest)
}

// NewRsaSigner creates a signer using RSA key
func NewRsaSigner(key *rsa.PrivateKey) ndn.CryptoSigner {
	return &rsaSigner{key}
}
