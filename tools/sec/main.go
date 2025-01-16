package sec

import (
	"github.com/named-data/ndnd/std/utils"
)

func GetSecCmdTree() utils.CmdTree {
	return utils.CmdTree{
		Name: "sec",
		Help: "NDN Security Utilities",
		Sub: []*utils.CmdTree{{
			Name: "keygen",
			Help: "Generate a new NDN key pair",
			Fun:  keygen,
		}, {
			// separator
		}, {
			Name: "keychain list",
			Help: "List keys in a keychain",
			Fun:  keychainList,
		}, {
			Name: "keychain import",
			Help: "Import keys or certs to a keychain",
			Fun:  keychainImport,
		}, {
			Name: "keychain get-key",
			Help: "Get a key from a keychain",
			Fun:  keychainGetKey,
		}, {
			Name: "keychain get-cert",
			Help: "Get a certificate from a keychain",
			Fun:  keychainGetCert,
		}, {
			// separator
		}, {
			Name: "pem-encode",
			Help: "Encode an NDN data to PEM representation",
			Fun:  pemEncode,
		}, {
			Name: "pem-decode",
			Help: "Decode a PEM representation of an NDN data",
			Fun:  pemDecode,
		}},
	}
}
