//go:build js && wasm

package keychain

import (
	"crypto/sha256"
	"encoding/hex"
	"syscall/js"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	sig "github.com/named-data/ndnd/std/security/signer"
	"github.com/named-data/ndnd/std/utils"
)

// KeyChainJS is a JS-based keychain.
type KeyChainJS struct {
	mem ndn.KeyChain
	api js.Value
}

// NewKeyChainJS creates a new JS-based keychain.
func NewKeyChainJS(api js.Value, pubStore ndn.Store) (ndn.KeyChain, error) {
	kc := &KeyChainJS{
		mem: NewKeyChainMem(pubStore),
		api: api,
	}

	list, err := utils.Await(api.Call("list"))
	if err != nil {
		return nil, err
	}

	list.Call("forEach", js.FuncOf(func(this js.Value, args []js.Value) any {
		size := args[0].Get("length").Int()
		content := make([]byte, size)
		js.CopyBytesToGo(content, args[0])

		err := InsertFile(kc.mem, content)
		if err != nil {
			log.Error(kc, "Failed to insert keychain entry", "err", err)
		}

		return nil
	}))

	return kc, nil
}

func (kc *KeyChainJS) String() string {
	return "keychain-js"
}

func (kc *KeyChainJS) Store() ndn.Store {
	return kc.mem.Store()
}

func (kc *KeyChainJS) Identities() []ndn.KeyChainIdentity {
	return kc.mem.Identities()
}

func (kc *KeyChainJS) IdentityByName(name enc.Name) ndn.KeyChainIdentity {
	return kc.mem.IdentityByName(name)
}

func (kc *KeyChainJS) InsertKey(signer ndn.Signer) error {
	err := kc.mem.InsertKey(signer)
	if err != nil {
		return err
	}

	secret, err := sig.MarshalSecret(signer)
	if err != nil {
		return err
	}

	return kc.writeFile(secret.Join(), EXT_KEY)
}

func (kc *KeyChainJS) InsertCert(wire []byte) error {
	err := kc.mem.InsertCert(wire)
	if err != nil {
		return err
	}

	return kc.writeFile(wire, EXT_CERT)
}

func (kc *KeyChainJS) writeFile(wire []byte, ext string) error {
	hash := sha256.Sum256(wire)
	filename := hex.EncodeToString(hash[:])

	jsBytes := js.Global().Get("Uint8Array").New(len(wire))
	js.CopyBytesToJS(jsBytes, wire)

	kc.api.Call("write", filename+ext, jsBytes)

	return nil
}
