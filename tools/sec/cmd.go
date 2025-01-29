package sec

import (
	"github.com/spf13/cobra"
)

var CmdSec = &cobra.Command{
	Use:   "sec",
	Short: "NDN Security Utilities",
}

func init() {
	// Key
	CmdSec.AddGroup(&cobra.Group{
		ID:    "key",
		Title: "Key Management",
	})

	// Keygen
	CmdSec.AddCommand(&cobra.Command{
		GroupID: "key",
		Use:     "keygen name key-type [params]",
		Short:   "Generate a new NDN key pair",
		Args:    cobra.MinimumNArgs(2),
		Example: `  ndnd sec keygen /alice ed25519
  ndnd sec keygen /bob/prefix rsa 2048
  ndnd sec keygen /carol ecc secp256r1`,
		Run: keygen,
	})

	// SignCert
	signCertCmd := &cobra.Command{
		GroupID: "key",
		Use:     "sign-cert key-file",
		Short:   "Sign a new NDN certificate",
		Long: `Sign a new NDN certificate

Expects CSR input on stdin.
Signer key and CSR can be TLV or PEM encoded.

CSR can be either a self-signed certificate or a secret key.
To generate a self-signed certificate, provide the same key
file as both the signer key and the CSR.`,
		Args: cobra.ExactArgs(1),
		Example: `  ndnd sec sign-cert alice.key < alice.key > alice.cert
  ndnd sec sign-cert alice.key --issuer ALICE < bob.csr > bob.cert`,
		Run: signCert,
	}
	signCertCmd.Flags().StringVar(&signCertFlags.Start, "start", "now", "Validity start time in YYYYMMDDhhmmss format")
	signCertCmd.Flags().StringVar(&signCertFlags.End, "end", "now + 1 year", "Validity end time in YYYYMMDDhhmmss format")
	signCertCmd.Flags().StringVar(&signCertFlags.Info, "info", "", "Additional info to be included in the certificate")
	signCertCmd.Flags().StringVar(&signCertFlags.Issuer, "issuer", "NA", "Issuer ID to be included in the certificate name")
	CmdSec.AddCommand(signCertCmd)

	// Keychain
	CmdSec.AddGroup(&cobra.Group{
		ID:    "keychain",
		Title: "Keychain Management",
	})

	// Keychain List
	CmdSec.AddCommand(&cobra.Command{
		GroupID: "keychain",
		Use:     "key-list keychain-uri",
		Short:   "List keys in a keychain",
		Run:     keychainList,
		Args:    cobra.ExactArgs(1),
		Example: `  ndnd sec key-list dir:///safe/keys`,
	})

	// Keychain Import
	CmdSec.AddCommand(&cobra.Command{
		GroupID: "keychain",
		Use:     "key-import keychain-uri",
		Short:   "Import keys or certs to a keychain",
		Long: `Import keys or certs to a keychain.

Expects one (TLV) or more (PEM) keys or certificates on stdin
and inserts them into the specified keychain.`,
		Args:    cobra.ExactArgs(1),
		Example: `  ndnd sec key-import dir:///safe/keys < alice.key`,
		Run:     keychainImport,
	})

	// Keychain GetKey
	CmdSec.AddCommand(&cobra.Command{
		GroupID: "keychain",
		Use:     "key-export keychain-uri key-name",
		Short:   "Export a key from a keychain",
		Long: `Export the specified key from a keychain.
If no KEY is specified, name will be treated as an identity
and the default key of the identity will be exported.`,
		Args:    cobra.ExactArgs(2),
		Example: `  ndnd sec key-export dir:///safe/keys /alice`,
		Run:     keychainExport,
	})

	// Encoding
	CmdSec.AddGroup(&cobra.Group{
		ID:    "encoding",
		Title: "Encoding Utilities",
	})

	// PEM Encode
	CmdSec.AddCommand(&cobra.Command{
		GroupID: "encoding",
		Use:     "pem-encode",
		Short:   "Encode an NDN key or cert to PEM",
		Long: `Encode a TLV NDN Key or Certificate to PEM.
Provide TLV data as input to stdin.`,
		Example: `  ndnd sec pem-encode < alice.tlv > alice.pem`,
		Args:    cobra.NoArgs,
		Run:     pemEncode,
	})

	// PEM Decode
	CmdSec.AddCommand(&cobra.Command{
		GroupID: "encoding",
		Use:     "pem-decode",
		Short:   "Decode PEM to NDN TLV format",
		Long: `Decode a PEM file containing a single NDN TLV.
Provide PEM data as input to stdin.`,
		Example: `  ndnd sec pem-decode < alice.pem > alice.tlv`,
		Args:    cobra.NoArgs,
		Run:     pemDecode,
	})
}
