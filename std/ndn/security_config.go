package ndn

import enc "github.com/named-data/ndnd/std/encoding"

// TrustConfig is the configuration of the trust module.
type TrustConfig struct {
	// KeyChain is the keychain.
	KeyChain KeyChain
	// Schema is the trust schema.
	Schema TrustSchema
	// Roots are the full names of the trust anchors.
	Roots []enc.Name
}

// Suggest suggests a signer for a given name.
func (tc *TrustConfig) Suggest(name enc.Name) Signer {
	return tc.Schema.Suggest(name, tc.KeyChain)
}
