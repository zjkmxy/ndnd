package keychain

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	spec "github.com/named-data/ndnd/std/ndn/spec_2022"
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
		if entry.IsDir() {
			continue
		}

		filename := filepath.Join(path, entry.Name())
		content, err := os.ReadFile(filename)
		if err != nil {
			log.Warn(kc, "Failed to read keychain entry", "file", filename, "error", err)
			continue
		}

		if len(content) == 0 {
			log.Warn(kc, "Empty keychain entry", "file", filename)
			continue
		}

		var wires [][]byte
		if content[0] == 0x06 { // raw data
			wires = append(wires, content)
		} else { // try text
			wires = security.TxtParse(content)
			if len(wires) == 0 {
				log.Warn(kc, "Failed to parse keychain entry", "file", filename)
				continue
			}
		}

		for _, wire := range wires {
			data, _, err := spec.Spec{}.ReadData(enc.NewBufferReader(wire))
			if err != nil {
				log.Warn(kc, "Failed to parse keychain entry", "file", filename, "error", err)
				continue
			}

			if data.ContentType() == nil {
				log.Warn(kc, "Entry with missing content type", "file", filename)
				continue
			}

			switch *data.ContentType() {
			case ndn.ContentTypeKey: // cert
				if err := kc.mem.InsertCert(wire); err != nil {
					log.Warn(kc, "Failed to insert certificate", "file", filename, "error", err)
				}
			case ndn.ContentTypeSecret: // key
				key, err := DecodeSecret(data)
				if err != nil || key == nil {
					log.Warn(kc, "Failed to decode key", "file", filename, "error", err)
					return nil, err
				}
				if err := kc.mem.InsertKey(key); err != nil {
					log.Warn(kc, "Failed to insert key", "file", filename, "error", err)
				}
			default:
				log.Warn(kc, "Unknown content type", "file", filename, "type", *data.ContentType())
			}
		}
	}

	return kc, nil
}

func (kc *KeyChainDir) String() string {
	return fmt.Sprintf("KeyChainDir (%s)", kc.path)
}

func (kc *KeyChainDir) GetIdentity(name enc.Name) ndn.Identity {
	return kc.mem.GetIdentity(name)
}

func (kc *KeyChainDir) InsertKey(signer ndn.Signer) error {
	// TODO: write to disk
	return kc.mem.InsertKey(signer)
}

func (kc *KeyChainDir) InsertCert(wire []byte) error {
	// TODO: write to disk
	return kc.mem.InsertCert(wire)
}
