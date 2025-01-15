package crypto_test

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"testing"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/ndn/spec_2022"
	"github.com/named-data/ndnd/std/security/crypto"
	"github.com/named-data/ndnd/std/utils"
	"github.com/stretchr/testify/require"
)

func TestEd25519SignerBasic(t *testing.T) {
	utils.SetTestingT(t)

	keyName, _ := enc.NameFromStr("/KEY")
	edkeybits := ed25519.NewKeyFromSeed([]byte("01234567890123456789012345678901"))
	signer := crypto.NewEd25519Signer(edkeybits, keyName)

	require.Equal(t, uint(ed25519.SignatureSize), signer.EstimateSize())
	require.Equal(t, ndn.SignatureEd25519, signer.Type())

	dataVal := enc.Wire{
		[]byte("\x07\x14\x08\x05local\x08\x03ndn\x08\x06prefix"),
		[]byte("\x14\x03\x18\x01\x00"),
	}
	sigValue := utils.WithoutErr(signer.Sign(dataVal))

	// For basic test, we use ed25519.Verify to verify the signature.
	require.True(t, ed25519.Verify(crypto.Ed25519DerivePubKey(edkeybits), dataVal.Join(), sigValue))
}

// TestEd25519SignerCertificate tests the validator using a given certificate for interoperability.
func TestEd25519SignerCertificate(t *testing.T) {
	utils.SetTestingT(t)

	const TestCert = `
Bv0BCgc1CAxFZDI1NTE5LWRlbW8IA0tFWQgQNWE2MTVkYjdjZjA2MDNiNQgEc2Vs
ZjYIAAABgQD8AY0UCRgBAhkEADbugBUsMCowBQYDK2VwAyEAQxUZBL+3I3D4oDIJ
tJvuCTguHM7AUbhlhA/wu8ZhrkwWVhsBBRwnByUIDEVkMjU1MTktZGVtbwgDS0VZ
CBA1YTYxNWRiN2NmMDYwM2I1/QD9Jv0A/g8xOTcwMDEwMVQwMDAwMDD9AP8PMjAy
MjA1MjZUMTUyODQ0F0DAAWCZzxQSCAV0tluFDry5aT1b+EgoYgT1JKxbKVb/tINx
M43PFy/2hDe8j61PuYD9tCah0TWapPwfXWi3fygA`
	spec := spec_2022.Spec{}

	certWire := utils.WithoutErr(base64.RawStdEncoding.DecodeString(TestCert))
	certData, covered, err := spec.ReadData(enc.NewBufferReader(certWire))
	require.NoError(t, err)

	pubKeyBits := utils.WithoutErr(x509.ParsePKIXPublicKey(certData.Content().Join()))
	pubKey := pubKeyBits.(ed25519.PublicKey)
	if pubKey == nil {
		require.Fail(t, "unexpected public key type")
	}
	require.True(t, crypto.Ed25519Validate(covered, certData.Signature(), pubKey))
}
