package keychain

import (
	"crypto/rand"

	enc "github.com/named-data/ndnd/std/encoding"
)

func MakeKeyName(name enc.Name) enc.Name {
	keyId := make([]byte, 8)
	rand.Read(keyId)

	return name.Append(
		enc.NewStringComponent(enc.TypeGenericNameComponent, "KEY"),
		enc.NewBytesComponent(enc.TypeGenericNameComponent, keyId),
	)
}
