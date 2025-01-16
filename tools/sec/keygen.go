package sec

import (
	"crypto/elliptic"
	"flag"
	"fmt"
	"os"
	"strconv"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/security"
	"github.com/named-data/ndnd/std/security/signer"
)

func keygen(args []string) {
	flagset := flag.NewFlagSet("keychain-import", flag.ExitOnError)
	flagset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <identity> <key-type> [params]\n", args[0])
		fmt.Fprintf(os.Stderr, "    key-type: rsa|ecc|ed25519\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Generates a new key pair and outputs the secret key in PEM.\n")
		fmt.Fprintf(os.Stderr, "The key pair is associated with the specified identity.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Example: %s /ndn/alice ed25519\n", args[0])
		flagset.PrintDefaults()
	}
	flagset.Parse(args[1:])

	argIdentity, argKeyType := flagset.Arg(0), flagset.Arg(1)
	if argIdentity == "" || argKeyType == "" {
		flagset.Usage()
		os.Exit(2)
	}

	name, err := enc.NameFromStr(argIdentity)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid identity: %s\n", argIdentity)
		os.Exit(1)
		return
	}

	name = security.MakeKeyName(name)
	sgn := keygenType(args[3:], name, argKeyType)

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

func keygenType(args []string, name enc.Name, keyType string) ndn.Signer {
	switch keyType {
	case "rsa":
		return keygenRsa(args, name)
	case "ed25519":
		return keygenEd25519(args, name)
	case "ecc":
		return keygecEcc(args, name)
	default:
		fmt.Fprintf(os.Stderr, "Unsupported key type: %s\n", keyType)
		os.Exit(2)
		return nil
	}
}

func keygenRsa(args []string, name enc.Name) ndn.Signer {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: keygen rsa <key-size>\n")
		os.Exit(2)
		return nil
	}

	keySize, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid key size: %s\n", args[0])
		os.Exit(1)
		return nil
	}

	sgn, err := signer.KeygenRsa(name, keySize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate RSA key: %s\n", err)
		os.Exit(1)
		return nil
	}

	return sgn
}

func keygenEd25519(_ []string, name enc.Name) ndn.Signer {
	sgn, err := signer.KeygenEd25519(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate Ed25519 key: %s\n", err)
		os.Exit(1)
		return nil
	}
	return sgn
}

func keygecEcc(args []string, name enc.Name) ndn.Signer {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: keygen ecc <curve>\n")
		fmt.Fprintf(os.Stderr, "Supported curves: secp256r1, secp384r1, secp521r1\n")
		os.Exit(2)
		return nil
	}

	var curve elliptic.Curve
	switch args[0] {
	case "secp256r1":
		curve = elliptic.P256()
	case "secp384r1":
		curve = elliptic.P384()
	case "secp521r1":
		curve = elliptic.P521()
	default:
		fmt.Fprintf(os.Stderr, "Unsupported curve: %s\n", args[0])
		os.Exit(1)
		return nil
	}

	sgn, err := signer.KeygenEcc(name, curve)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate EC key: %s\n", err)
		os.Exit(1)
		return nil
	}
	return sgn
}
