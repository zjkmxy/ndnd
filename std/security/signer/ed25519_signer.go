package signer

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

// ed25519Signer is a signer that uses Ed25519 key to sign packets.
type ed25519Signer struct {
	name enc.Name
	key  ed25519.PrivateKey
}

func (s *ed25519Signer) Type() ndn.SigType {
	return ndn.SignatureEd25519
}

func (s *ed25519Signer) KeyName() enc.Name {
	return s.name
}

func (s *ed25519Signer) KeyLocator() enc.Name {
	return s.name
}

func (s *ed25519Signer) EstimateSize() uint {
	return ed25519.SignatureSize
}

func (s *ed25519Signer) Sign(covered enc.Wire) ([]byte, error) {
	return ed25519.Sign(s.key, covered.Join()), nil
}

func (s *ed25519Signer) Public() ([]byte, error) {
	return s.key[ed25519.PublicKeySize:], nil
}

func (s *ed25519Signer) Secret() ([]byte, error) {
	return s.key, nil
}

// NewEd25519Signer creates a signer using ed25519 key
func NewEd25519Signer(name enc.Name, key ed25519.PrivateKey) ndn.Signer {
	return &ed25519Signer{name, key}
}

// Ed25519Keygen creates a signer using a new Ed25519 key
func KeygenEd25519(name enc.Name) (ndn.Signer, error) {
	_, sk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return NewEd25519Signer(name, sk), nil
}

// ParseEd25519 parses a signer from a byte slice.
func ParseEd25519(name enc.Name, key []byte) (ndn.Signer, error) {
	if len(key) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid Ed25519 private key size")
	}
	return NewEd25519Signer(name, key), nil
}
