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
			Help: "List keys in the specified keychain",
			Fun:  keychainList,
		}, {
			Name: "keychain import",
			Help: "Import a key or certificate to the specified keychain",
			Fun:  keychainImport,
		}, {
			Name: "keychain export-key",
			Help: "Export a keyfrom the specified keychain",
			Fun:  keychainExportKey,
		}, {
			Name: "keychain export-cert",
			Help: "Export a certificate from the specified keychain",
			Fun:  keychainExportCert,
		}, {
			// separator
		}, {
			Name: "txt-from",
			Help: "Convert an NDN data to text representation",
			Fun:  txtFrom,
		}, {
			Name: "txt-parse",
			Help: "Parse a text representation of an NDN data",
			Fun:  txtParse,
		}},
	}
}
