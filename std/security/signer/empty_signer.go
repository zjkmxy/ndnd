package signer

import (
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

// emptySigner is a signer used for test only. It gives an empty signature value.
type emptySigner struct{}

func (emptySigner) Type() ndn.SigType {
	return ndn.SignatureEmptyTest
}

func (emptySigner) KeyName() enc.Name {
	return nil
}

func (emptySigner) KeyLocator() enc.Name {
	return nil
}

func (emptySigner) EstimateSize() uint {
	return 0
}

func (emptySigner) Sign(covered enc.Wire) ([]byte, error) {
	return []byte{}, nil
}

func (emptySigner) Public() ([]byte, error) {
	return nil, ndn.ErrNoPubKey
}

// NewEmptySigner creates an empty signer for test.
func NewEmptySigner() ndn.Signer {
	return emptySigner{}
}
