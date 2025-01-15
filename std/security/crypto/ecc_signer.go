package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

// EccSigner is a signer that uses ECC key to sign packets.
type EccSigner struct {
	name   enc.Name
	key    *ecdsa.PrivateKey
	keyLen uint
}

func (s *EccSigner) Type() ndn.SigType {
	return ndn.SignatureSha256WithEcdsa
}

func (s *EccSigner) KeyLocator() enc.Name {
	return s.name
}

func (s *EccSigner) EstimateSize() uint {
	return s.keyLen
}

func (s *EccSigner) Sign(covered enc.Wire) ([]byte, error) {
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

func (s *EccSigner) Public() ([]byte, error) {
	return x509.MarshalPKIXPublicKey(&s.key.PublicKey)
}

func (s *EccSigner) Secret() ([]byte, error) {
	return x509.MarshalECPrivateKey(s.key)
}

// NewEccSigner creates a signer using ECDSA key
func NewEccSigner(name enc.Name, key *ecdsa.PrivateKey) ndn.Signer {
	keyLen := (key.Curve.Params().BitSize*2 + 7) / 8
	keyLen += keyLen%2 + 8
	return &EccSigner{
		name:   name,
		key:    key,
		keyLen: uint(keyLen),
	}
}

// KeygenEcc creates a signer using a new ECDSA key
func KeygenEcc(name enc.Name, curve elliptic.Curve) (ndn.Signer, error) {
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, enc.ErrUnexpected{Err: err}
	}
	return NewEccSigner(name, key), nil
}

// ParseEcc parses a signer from a byte slice.
func ParseEcc(name enc.Name, key []byte) (ndn.Signer, error) {
	sk, err := x509.ParseECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	return NewEccSigner(name, sk), nil
}
