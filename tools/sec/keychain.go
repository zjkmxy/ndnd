package sec

import (
	"flag"
	"fmt"
	"io"
	"os"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/object"
	"github.com/named-data/ndnd/std/security"
	"github.com/named-data/ndnd/std/security/keychain"
	sig "github.com/named-data/ndnd/std/security/signer"
)

func keychainList(args []string) {
	flagset := flag.NewFlagSet("keychain-list", flag.ExitOnError)
	flagset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <keychain>\n", args[0])
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Lists all keys in the specified keychain.\n")
		fmt.Fprintf(os.Stderr, "Target <keychain> is specified as a URI.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Example: %s dir:///safe/keys\n", args[0])
		flagset.PrintDefaults()
	}
	flagset.Parse(args[1:])

	argKeychain := flagset.Arg(0)
	if argKeychain == "" {
		flagset.Usage()
		os.Exit(2)
	}

	kc, err := keychain.NewKeyChain(argKeychain, object.NewMemoryStore())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open keychain: %s\n", err)
		os.Exit(1)
		return
	}

	for _, id := range kc.GetIdentities() {
		fmt.Printf("%s\n", id.Name())
		for _, key := range id.Keys() {
			fmt.Printf("==> %s\n", key.KeyName())
		}
	}
}

func keychainImport(args []string) {
	flagset := flag.NewFlagSet("keychain-import", flag.ExitOnError)
	flagset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <keychain>\n", args[0])
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Expects one (TLV) or more (PEM) keys or certificates on stdin\n")
		fmt.Fprintf(os.Stderr, "and inserts them into the specified keychain.\n")
		fmt.Fprintf(os.Stderr, "Target <keychain> is specified as a URI.\n")
		flagset.PrintDefaults()
	}
	flagset.Parse(args[1:])

	argKeychain := flagset.Arg(0)
	if argKeychain == "" {
		flagset.Usage()
		os.Exit(2)
	}

	kc, err := keychain.NewKeyChain(argKeychain, object.NewMemoryStore())
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
	flagset := flag.NewFlagSet("keychain-get-key", flag.ExitOnError)
	flagset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <keychain> <name>\n", args[0])
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Exports the specified key from the specified keychain.\n")
		fmt.Fprintf(os.Stderr, "If no KEY is specified, name will be treated as an identity\n")
		fmt.Fprintf(os.Stderr, "and the default key of the identity will be exported.\n")
		fmt.Fprintf(os.Stderr, "Target <keychain> is specified as a URI.\n")
		flagset.PrintDefaults()
	}
	flagset.Parse(args[1:])

	argKeychain, argName := flagset.Arg(0), flagset.Arg(1)
	if argKeychain == "" || argName == "" {
		flagset.Usage()
		os.Exit(2)
	}

	name, err := enc.NameFromStr(argName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid key name: %s\n", argName)
		os.Exit(1)
		return
	}

	kc, err := keychain.NewKeyChain(argKeychain, object.NewMemoryStore())
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

	var signer ndn.Signer
	if keyName == nil {
		if len(idObj.Keys()) > 0 {
			signer = idObj.Keys()[0].Signer()
		}
	} else {
		for _, key := range idObj.Keys() {
			if key.KeyName().Equal(keyName) {
				signer = key.Signer()
				break
			}
		}
	}
	if signer == nil {
		fmt.Fprintf(os.Stderr, "Key not found: %s\n", keyName)
		os.Exit(1)
		return
	}

	secret, err := sig.MarshalSecret(signer)
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
	panic("not implemented")
}
