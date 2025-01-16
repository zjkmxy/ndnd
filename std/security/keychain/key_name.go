package keychain

import (
	"crypto/rand"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

func MakeKeyName(name enc.Name) enc.Name {
	keyId := make([]byte, 8)
	rand.Read(keyId)

	return name.Append(
		enc.NewStringComponent(enc.TypeGenericNameComponent, "KEY"),
		enc.NewBytesComponent(enc.TypeGenericNameComponent, keyId),
	)
}

func GetIdentityFromKeyName(name enc.Name) (enc.Name, error) {
	if len(name) < 3 {
		return nil, ndn.ErrInvalidValue{Item: "key name"}
	}
	if name[len(name)-2].String() != "KEY" {
		return nil, ndn.ErrInvalidValue{Item: "KEY component"}
	}

	return name[:len(name)-2], nil
}
