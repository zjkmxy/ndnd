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
	"github.com/named-data/ndnd/std/security/signer"
)

func keychainList(args []string) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s sec keychain list <keychain>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s sec keychain list dir:///safe/keys\n", os.Args[0])
		os.Exit(2)
		return
	}

	uri := args[1]
	kc, err := keychain.NewKeyChain(uri, object.NewMemoryStore())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open keychain: %s\n", err)
		os.Exit(1)
		return
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
		os.Exit(2)
		return
	}

	uri := args[1]
	kc, err := keychain.NewKeyChain(uri, object.NewMemoryStore())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open keychain: %s\n", err)
		os.Exit(1)
		return
	}

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input: %s\n", err)
		os.Exit(1)
		return
	}

	err = keychain.InsertFile(kc, input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to insert keychain entries: %s\n", err)
		os.Exit(1)
		return
	}
}

func keychainGetKey(args []string) {
	if len(args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s sec keychain get-key <keychain> <key-name>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "    if no KEY is specified, export the default identity key\n")
		os.Exit(1)
		return
	}

	name, err := enc.NameFromStr(args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid key name: %s\n", args[2])
		os.Exit(1)
		return
	}

	uri := args[1]
	kc, err := keychain.NewKeyChain(uri, object.NewMemoryStore())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open keychain: %s\n", err)
		os.Exit(1)
		return
	}

	keyName := name
	id, err := security.GetIdentityFromKeyName(name)
	if err != nil { // not a key name
		id = name
		keyName = nil
	}

	idObj := kc.GetIdentity(id)
	if idObj == nil {
		fmt.Fprintf(os.Stderr, "Identity not found: %s\n", id)
		os.Exit(1)
		return
	}

	var sgn ndn.Signer
	if keyName == nil {
		sgn = idObj.Signer()
	} else {
		for _, key := range idObj.AllSigners() {
			if key.KeyName().Equal(keyName) {
				sgn = key
				break
			}
		}
	}
	if sgn == nil {
		fmt.Fprintf(os.Stderr, "Key not found: %s\n", keyName)
		os.Exit(1)
		return
	}

	secret, err := signer.EncodeSecret(sgn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to encode secret key: %s\n", err)
		os.Exit(1)
		return
	}

	out, err := security.PemEncode(secret.Join())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to convert secret key to text: %s\n", err)
		os.Exit(1)
		return
	}

	os.Stdout.Write(out)
}

func keychainGetCert(args []string) {
}
