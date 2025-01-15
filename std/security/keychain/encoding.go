package keychain

import (
	"encoding/base64"
	"fmt"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	spec "github.com/named-data/ndnd/std/ndn/spec_2022"
	"github.com/named-data/ndnd/std/security/crypto"
	"github.com/named-data/ndnd/std/utils"
)

// EncodeSecret encodes a key secret to a signed NDN Data packet.
func EncodeSecret(key ndn.Signer) (enc.Wire, error) {
	// Get key name
	name := key.KeyLocator()
	if name == nil {
		return nil, ndn.ErrInvalidValue{Item: "key locator"}
	}

	// Get key secret bits
	var sk []byte = nil
	var err error = nil
	switch key := key.(type) {
	case *crypto.EccSigner:
		sk, err = key.Secret()
	case *crypto.RsaSigner:
		sk, err = key.Secret()
		fmt.Println(base64.StdEncoding.EncodeToString(sk))
	case *crypto.Ed25519Signer:
		sk, err = key.Secret()
	default:
		return nil, ndn.ErrNotSupported{Item: "key type"}
	}
	if err != nil {
		return nil, err
	} else if sk == nil {
		return nil, ndn.ErrInvalidValue{Item: "key secret"}
	}

	// Encode key data packet
	cfg := &ndn.DataConfig{
		ContentType: utils.IdPtr(ndn.ContentTypeSecret),
	}
	data, err := spec.Spec{}.MakeData(name, cfg, enc.Wire{sk}, key)
	if err != nil {
		return nil, err
	}

	return data.Wire, nil
}

// DecodeSecret decodes a signed NDN Data packet to a key secret.
func DecodeSecret(data ndn.Data) (ndn.Signer, error) {
	// Check data content type
	if data.ContentType() == nil || *data.ContentType() != ndn.ContentTypeSecret {
		return nil, ndn.ErrInvalidValue{Item: "content type"}
	}

	// Check signature is present
	if data.Signature() == nil {
		return nil, ndn.ErrInvalidValue{Item: "signature"}
	}

	// Decode key secret bits
	wire := data.Content().Join()
	if len(wire) == 0 {
		return nil, ndn.ErrInvalidValue{Item: "content"}
	}

	// Decode key secret depending on signature type
	switch data.Signature().SigType() {
	case ndn.SignatureSha256WithEcdsa:
		return crypto.ParseEcc(data.Name(), wire)
	case ndn.SignatureSha256WithRsa:
		return crypto.ParseRsa(data.Name(), wire)
	case ndn.SignatureEd25519:
		return crypto.ParseEd25519(data.Name(), wire)
	default:
		return nil, ndn.ErrNotSupported{Item: "signature type"}
	}
}
