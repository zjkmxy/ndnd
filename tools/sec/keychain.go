package sec

import (
	"fmt"
	"io"
	"os"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/object"
	"github.com/named-data/ndnd/std/security"
	"github.com/named-data/ndnd/std/security/keychain"
)

func keychainList(args []string) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s sec keychain list <keychain>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s sec keychain list dir:///safe/keys\n", os.Args[0])
		return
	}

	uri := args[1]
	kc, err := keychain.NewKeyChain(uri, object.NewMemoryStore())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open keychain: %s\n", err)
		os.Exit(1)
	}

	for _, id := range kc.GetIdentities() {
		fmt.Printf("%s\n", id.Name())
		for _, key := range id.AllSigners() {
			fmt.Printf("==> %s\n", key.KeyName())
		}
	}
}

func keychainImport(args []string) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s sec keychain import <keychain>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "   expects import data on stdin\n")
		return
	}

	uri := args[1]
	kc, err := keychain.NewKeyChain(uri, object.NewMemoryStore())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open keychain: %s\n", err)
		os.Exit(1)
	}

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input: %s\n", err)
		os.Exit(1)
	}

	err = keychain.InsertFile(kc, input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to insert keychain entries: %s\n", err)
		os.Exit(1)
	}
}

func keychainExportKey(args []string) {
	if len(args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s sec keychain export <keychain> <key-name>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "    if not KEY is specified, export the default key for the identity\n")
		return
	}

	name, err := enc.NameFromStr(args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid key name: %s\n", args[2])
		os.Exit(1)
	}

	uri := args[1]
	kc, err := keychain.NewKeyChain(uri, object.NewMemoryStore())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open keychain: %s\n", err)
		os.Exit(1)
	}

	keyName := name
	id, err := security.GetIdentityFromKeyName(name)
	if err != nil {
		id = name
		keyName = nil
	}

	idObj := kc.GetIdentity(id)
	if idObj == nil {
		fmt.Fprintf(os.Stderr, "Identity not found: %s\n", id)
		os.Exit(1)
	}

	var signer ndn.Signer
	if keyName == nil {
		signer = idObj.Signer()
	} else {
		for _, key := range idObj.AllSigners() {
			if key.KeyName().Equal(keyName) {
				signer = key
				break
			}
		}
	}
	if signer == nil {
		fmt.Fprintf(os.Stderr, "Key not found: %s\n", keyName)
		os.Exit(1)
	}

	secret, err := keychain.EncodeSecret(signer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to encode secret key: %s\n", err)
		os.Exit(1)
	}

	out, err := security.TxtFrom(secret.Join())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to convert secret key to text: %s\n", err)
		os.Exit(1)
	}

	os.Stdout.Write(out)
}

func keychainExportCert(args []string) {
}
