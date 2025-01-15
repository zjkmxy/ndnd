package security

import (
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/security/crypto"

	"github.com/named-data/ndnd/std/ndn"
)

type signerStatic struct {
	ndn.CryptoSigner
}

func (signerStatic) KeyLocator() enc.Name {
	return nil
}

func NewNullSigner() ndn.Signer {
	return &signerStatic{CryptoSigner: crypto.NewNullSigner()}
}

func NewSha256Signer() ndn.Signer {
	return &signerStatic{CryptoSigner: crypto.NewSha256Signer()}
}

func NewHmacSigner(key []byte) ndn.Signer {
	return &signerStatic{CryptoSigner: crypto.NewHmacSigner(key)}
}
