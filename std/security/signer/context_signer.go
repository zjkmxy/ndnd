package signer

import (
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

// ContextSigner is a wrapper around a signer to provide extra context.
type ContextSigner struct {
	Signer         ndn.Signer
	KeyLocatorName enc.Name
}

func (s *ContextSigner) Type() ndn.SigType {
	return s.Signer.Type()
}

func (s *ContextSigner) KeyName() enc.Name {
	return s.Signer.KeyName()
}

func (s *ContextSigner) KeyLocator() enc.Name {
	return s.KeyLocatorName
}

func (s *ContextSigner) EstimateSize() uint {
	return s.Signer.EstimateSize()
}

func (s *ContextSigner) Sign(covered enc.Wire) ([]byte, error) {
	return s.Signer.Sign(covered)
}

func (s *ContextSigner) Public() ([]byte, error) {
	return s.Signer.Public()
}
