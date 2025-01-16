package keychain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/security"
)

// KeyChainDir is a directory-based keychain.
type KeyChainDir struct {
	wmut sync.Mutex
	mem  ndn.KeyChain
	path string
}

// NewKeyChainDir creates a new in-memory keychain.
func NewKeyChainDir(path string, pubStore ndn.Store) (ndn.KeyChain, error) {
	kc := &KeyChainDir{
		wmut: sync.Mutex{},
		mem:  NewKeyChainMem(pubStore),
		path: path,
	}

	// Populate keychain from disk
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".key") &&
			!strings.HasSuffix(entry.Name(), ".cert") {
			continue
		}

		if entry.IsDir() {
			continue
		}

		filename := filepath.Join(path, entry.Name())
		content, err := os.ReadFile(filename)
		if err != nil {
			log.Warn(kc, "Failed to read keychain entry", "file", filename, "error", err)
			continue
		}

		err = InsertFile(kc.mem, content)
		if err != nil {
			log.Error(kc, "Failed to insert keychain entries", "file", filename, "error", err)
		}
	}

	return kc, nil
}

func (kc *KeyChainDir) String() string {
	return fmt.Sprintf("KeyChainDir (%s)", kc.path)
}

func (kc *KeyChainDir) GetIdentities() []ndn.Identity {
	return kc.mem.GetIdentities()
}

func (kc *KeyChainDir) GetIdentity(name enc.Name) ndn.Identity {
	return kc.mem.GetIdentity(name)
}

func (kc *KeyChainDir) InsertKey(signer ndn.Signer) error {
	err := kc.mem.InsertKey(signer)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(signer.KeyName().Bytes())
	fname := hex.EncodeToString(hash[:])
	path := filepath.Join(kc.path, fname+".key")

	kc.wmut.Lock()
	defer kc.wmut.Unlock()

	secret, err := EncodeSecret(signer)
	if err != nil {
		return err
	}

	txt, err := security.TxtFrom(secret.Join())
	if err != nil {
		return err
	}

	return os.WriteFile(path, txt, 0644)
}

func (kc *KeyChainDir) InsertCert(wire []byte) error {
	// TODO: write to disk
	return kc.mem.InsertCert(wire)
}
