package keychain

import (
	"sync"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	spec "github.com/named-data/ndnd/std/ndn/spec_2022"
	sec "github.com/named-data/ndnd/std/security"
)

// KeyChainMem is an in-memory keychain.
type KeyChainMem struct {
	mut        sync.RWMutex
	identities map[string]*identity
	pubStore   ndn.Store
}

// NewKeyChainMem creates a new in-memory keychain.
func NewKeyChainMem(pubStore ndn.Store) ndn.KeyChain {
	return &KeyChainMem{
		mut:        sync.RWMutex{},
		identities: make(map[string]*identity),
		pubStore:   pubStore,
	}
}

func (kc *KeyChainMem) GetIdentities() []ndn.Identity {
	kc.mut.RLock()
	defer kc.mut.RUnlock()
	ids := make([]ndn.Identity, 0, len(kc.identities))
	for _, id := range kc.identities {
		ids = append(ids, id)
	}
	return ids
}

func (kc *KeyChainMem) GetIdentity(name enc.Name) ndn.Identity {
	kc.mut.RLock()
	defer kc.mut.RUnlock()
	if id, ok := kc.identities[name.String()]; ok {
		return id
	}
	return nil
}

func (kc *KeyChainMem) InsertKey(signer ndn.Signer) error {
	kc.mut.Lock()
	defer kc.mut.Unlock()

	// Get key name
	id, err := sec.GetIdentityFromKeyName(signer.KeyName())
	if err != nil {
		return err
	}
	hash := id.String()

	// Insert signer
	idObj := kc.identities[hash]
	if idObj == nil {
		idObj = &identity{name: id}
		kc.identities[hash] = idObj
	}
	idObj.signers = append([]ndn.Signer{signer}, idObj.signers...)

	// TODO: fix sort order

	return nil
}

func (kc *KeyChainMem) String() string {
	return "keychain-mem"
}

func (kc *KeyChainMem) InsertCert(wire []byte) error {
	data, _, err := spec.Spec{}.ReadData(enc.NewBufferReader(wire))
	if err != nil {
		return err
	}

	if data.ContentType() == nil || *data.ContentType() != ndn.ContentTypeKey {
		return ndn.ErrInvalidValue{Item: "content type"}
	}

	// /<IdentityName>/KEY/<KeyId>/<IssuerId>/<Version>
	name := data.Name()
	if len(name) < 5 {
		return ndn.ErrInvalidValue{Item: "name length"}
	}

	keyComp := name[len(name)-4]
	if keyComp.String() != "KEY" {
		return ndn.ErrInvalidValue{Item: "KEY component"}
	}

	versionComp := name[len(name)-1]
	if versionComp.Typ != enc.TypeVersionNameComponent {
		return ndn.ErrInvalidValue{Item: "version component"}
	}
	version := versionComp.NumberVal()

	kc.pubStore.Put(name, version, wire)
	return nil
}
