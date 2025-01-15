package crypto

import (
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

// nullSigner is a signer used for test only. It gives an empty signature value.
type nullSigner struct{}

func (nullSigner) Type() ndn.SigType {
	return ndn.SignatureEmptyTest
}

func (nullSigner) KeyLocator() enc.Name {
	return nil
}

func (nullSigner) EstimateSize() uint {
	return 0
}

func (nullSigner) Sign(covered enc.Wire) ([]byte, error) {
	return []byte{}, nil
}

// NewNullSigner creates an empty signer for test.
func NewNullSigner() ndn.Signer {
	return nullSigner{}
}
