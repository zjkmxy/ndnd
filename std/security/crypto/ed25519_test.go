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

var TEST_KEY_NAME, _ = enc.NameFromStr("/KEY")

func testEd25519Verify(t *testing.T, signer ndn.Signer, verifyKey []byte) bool {
	require.Equal(t, uint(ed25519.SignatureSize), signer.EstimateSize())
	require.Equal(t, ndn.SignatureEd25519, signer.Type())
	require.Equal(t, TEST_KEY_NAME, signer.KeyName())

	dataVal := enc.Wire{
		[]byte("\x07\x14\x08\x05local\x08\x03ndn\x08\x06prefix"),
		[]byte("\x14\x03\x18\x01\x00"),
	}
	sigValue := utils.WithoutErr(signer.Sign(dataVal))

	// For basic test, we use ed25519.Verify to verify the signature.
	return ed25519.Verify(verifyKey, dataVal.Join(), sigValue)
}

func TestEd25519SignerNew(t *testing.T) {
	utils.SetTestingT(t)

	edkeybits := ed25519.NewKeyFromSeed([]byte("01234567890123456789012345678901"))
	signer := crypto.NewEd25519Signer(TEST_KEY_NAME, edkeybits)
	pub := utils.WithoutErr(signer.Public())
	require.True(t, testEd25519Verify(t, signer, pub))
}

func TestEd25519Keygen(t *testing.T) {
	utils.SetTestingT(t)

	signer1 := utils.WithoutErr(crypto.KeygenEd25519(TEST_KEY_NAME))
	pub1 := utils.WithoutErr(signer1.Public())
	require.True(t, testEd25519Verify(t, signer1, pub1))

	signer2 := utils.WithoutErr(crypto.KeygenEd25519(TEST_KEY_NAME))
	pub2 := utils.WithoutErr(signer2.Public())
	require.True(t, testEd25519Verify(t, signer2, pub2))

	// Check that the two signers are different.
	require.False(t, testEd25519Verify(t, signer2, pub1))
}

func TestEd25519Parse(t *testing.T) {
	utils.SetTestingT(t)

	edkeybits := ed25519.NewKeyFromSeed([]byte("01234567890123456789012345678901"))
	signer1 := crypto.NewEd25519Signer(TEST_KEY_NAME, edkeybits)

	secret := utils.WithoutErr(signer1.(*crypto.Ed25519Signer).Secret())
	signer2 := utils.WithoutErr(crypto.ParseEd25519(TEST_KEY_NAME, secret))

	// Check that the two signers are the same.
	pub1 := utils.WithoutErr(signer1.Public())
	require.True(t, testEd25519Verify(t, signer2, pub1))
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
