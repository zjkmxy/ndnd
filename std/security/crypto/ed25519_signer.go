package crypto

import (
	"crypto/ed25519"
	"errors"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

// ed25519Signer is a signer that uses Ed25519 key to sign packets.
type ed25519Signer struct {
	key  ed25519.PrivateKey
	name enc.Name
}

func (s *ed25519Signer) Type() ndn.SigType {
	return ndn.SignatureEd25519
}

func (s *ed25519Signer) KeyLocator() enc.Name {
	return s.name
}

func (s *ed25519Signer) EstimateSize() uint {
	return ed25519.SignatureSize
}

func (s *ed25519Signer) Sign(covered enc.Wire) ([]byte, error) {
	if len(s.key) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid Ed25519 private key size")
	}
	return ed25519.Sign(s.key, covered.Join()), nil
}

// NewEd25519Signer creates a signer using ed25519 key
func NewEd25519Signer(key ed25519.PrivateKey, name enc.Name) ndn.Signer {
	return &ed25519Signer{key, name}
}

// Ed25519DerivePubKey derives the public key from a private key.
func Ed25519DerivePubKey(privKey ed25519.PrivateKey) ed25519.PublicKey {
	return ed25519.PublicKey(privKey[ed25519.PublicKeySize:])
}
