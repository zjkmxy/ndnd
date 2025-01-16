package keychain

import (
	"errors"
	"net/url"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	spec "github.com/named-data/ndnd/std/ndn/spec_2022"
	"github.com/named-data/ndnd/std/security"
)

func NewKeyChain(uri string, pubStore ndn.Store) (ndn.KeyChain, error) {
	url, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	switch url.Scheme {
	case "mem":
		return NewKeyChainMem(pubStore), nil
	case "dir":
		return NewKeyChainDir(url.Path, pubStore)
	default:
		return nil, ndn.ErrInvalidValue{Item: "keychain-scheme", Value: url.Scheme}
	}
}

// InsertFile inserts all signers and certs from the given content.
func InsertFile(kc ndn.KeyChain, content []byte) error {
	if len(content) == 0 {
		return errors.New("empty keychain entry")
	}

	var wires [][]byte
	if content[0] == 0x06 { // raw data
		wires = append(wires, content)
	} else { // try text
		wires = security.PemDecode(content)
		if len(wires) == 0 {
			return errors.New("no valid keychain entry found")
		}
	}

	for _, wire := range wires {
		data, _, err := spec.Spec{}.ReadData(enc.NewBufferReader(wire))
		if err != nil {
			log.Warn(kc, "Failed to read keychain entry", "error", err)
			continue
		}

		if data.ContentType() == nil {
			log.Warn(kc, "No content type found", "name", data.Name())
			continue
		}

		switch *data.ContentType() {
		case ndn.ContentTypeKey: // cert
			if err := kc.InsertCert(wire); err != nil {
				log.Warn(kc, "Failed to insert certificate", "error", err)
				continue
			}

		case ndn.ContentTypeSecret: // key
			key, err := DecodeSecret(data)
			if err != nil || key == nil {
				log.Warn(kc, "Failed to decode key", "name", data.Name(), "error", err)
				continue
			}
			if err := kc.InsertKey(key); err != nil {
				log.Warn(kc, "Failed to insert key", "error", err)
				continue
			}

		default:
			log.Warn(kc, "Unknown content type", "name", data.Name(), "type", *data.ContentType())
		}
	}

	return nil
}
