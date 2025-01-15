package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

// eccSigner is a signer that uses ECC key to sign packets.
type eccSigner struct {
	key    *ecdsa.PrivateKey
	keyLen uint
	name   enc.Name
}

func (s *eccSigner) Type() ndn.SigType {
	return ndn.SignatureSha256WithEcdsa
}

func (s *eccSigner) KeyLocator() enc.Name {
	return s.name
}

func (s *eccSigner) EstimateSize() uint {
	return s.keyLen
}

func (s *eccSigner) Sign(covered enc.Wire) ([]byte, error) {
	h := sha256.New()
	for _, buf := range covered {
		_, err := h.Write(buf)
		if err != nil {
			return nil, enc.ErrUnexpected{Err: err}
		}
	}
	digest := h.Sum(nil)
	return ecdsa.SignASN1(rand.Reader, s.key, digest)
}

// NewEccSigner creates a signer using ECDSA key
func NewEccSigner(key *ecdsa.PrivateKey, name enc.Name) ndn.Signer {
	keyLen := (key.Curve.Params().BitSize*2 + 7) / 8
	keyLen += keyLen%2 + 8
	return &eccSigner{
		key:    key,
		keyLen: uint(keyLen),
		name:   name,
	}
}
