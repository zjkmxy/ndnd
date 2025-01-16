package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

// RsaSigner is a signer that uses ECC key to sign packets.
type RsaSigner struct {
	name enc.Name
	key  *rsa.PrivateKey
}

func (s *RsaSigner) Type() ndn.SigType {
	return ndn.SignatureSha256WithRsa
}

func (s *RsaSigner) KeyName() enc.Name {
	return s.name
}

func (s *RsaSigner) EstimateSize() uint {
	return uint(s.key.Size())
}

func (s *RsaSigner) Sign(covered enc.Wire) ([]byte, error) {
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

func (s *RsaSigner) Public() ([]byte, error) {
	return x509.MarshalPKIXPublicKey(&s.key.PublicKey)
}

func (s *RsaSigner) Secret() ([]byte, error) {
	return x509.MarshalPKCS1PrivateKey(s.key), nil
}

// NewRsaSigner creates a signer using RSA key
func NewRsaSigner(name enc.Name, key *rsa.PrivateKey) ndn.Signer {
	return &RsaSigner{name, key}
}

// KeygenRsa creates a signer using a new RSA key
func KeygenRsa(name enc.Name, size int) (ndn.Signer, error) {
	key, err := rsa.GenerateKey(rand.Reader, size)
	if err != nil {
		return nil, enc.ErrUnexpected{Err: err}
	}
	return NewRsaSigner(name, key), nil
}

// ParseRsa creates a signer using a RSA key bytes
func ParseRsa(name enc.Name, key []byte) (ndn.Signer, error) {
	sk, err := x509.ParsePKCS1PrivateKey(key)
	if err != nil {
		return nil, err
	}
	return NewRsaSigner(name, sk), nil
}
