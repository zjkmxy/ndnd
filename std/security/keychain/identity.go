package keychain

import (
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

type identity struct {
	// name is the name of the identity.
	name enc.Name
	// signers is the list of keys.
	signers []ndn.Signer
}

func (id *identity) Name() enc.Name {
	return id.name
}

func (id *identity) Signer() ndn.Signer {
	if len(id.signers) == 0 {
		return nil
	}
	return id.signers[0]
}

func (id *identity) AllSigners() []ndn.Signer {
	return id.signers
}
