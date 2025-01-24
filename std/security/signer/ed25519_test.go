package signer_test

import (
	"crypto/ed25519"
	"crypto/x509"
	"testing"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	sig "github.com/named-data/ndnd/std/security/signer"
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
	verifyKeyBits := utils.WithoutErr(x509.ParsePKIXPublicKey(verifyKey)).(ed25519.PublicKey)
	return ed25519.Verify(verifyKeyBits, dataVal.Join(), sigValue)
}

func TestEd25519SignerNew(t *testing.T) {
	utils.SetTestingT(t)

	edkeybits := ed25519.NewKeyFromSeed([]byte("01234567890123456789012345678901"))
	signer := sig.NewEd25519Signer(TEST_KEY_NAME, edkeybits)
	pub := utils.WithoutErr(signer.Public())
	require.True(t, testEd25519Verify(t, signer, pub))
}

func TestEd25519Keygen(t *testing.T) {
	utils.SetTestingT(t)

	signer1 := utils.WithoutErr(sig.KeygenEd25519(TEST_KEY_NAME))
	pub1 := utils.WithoutErr(signer1.Public())
	require.True(t, testEd25519Verify(t, signer1, pub1))

	signer2 := utils.WithoutErr(sig.KeygenEd25519(TEST_KEY_NAME))
	pub2 := utils.WithoutErr(signer2.Public())
	require.True(t, testEd25519Verify(t, signer2, pub2))

	// Check that the two signers are different.
	require.False(t, testEd25519Verify(t, signer2, pub1))
}

func TestEd25519Parse(t *testing.T) {
	utils.SetTestingT(t)

	edkeybits := ed25519.NewKeyFromSeed([]byte("01234567890123456789012345678901"))
	signer1 := sig.NewEd25519Signer(TEST_KEY_NAME, edkeybits)

	secret := utils.WithoutErr(sig.GetSecret(signer1))
	signer2 := utils.WithoutErr(sig.ParseEd25519(TEST_KEY_NAME, secret))

	// Check that the two signers are the same.
	pub1 := utils.WithoutErr(signer1.Public())
	require.True(t, testEd25519Verify(t, signer2, pub1))

	// Make sure parse fails with the public key.
	pub2 := utils.WithoutErr(signer1.Public())
	_, err := sig.ParseEd25519(TEST_KEY_NAME, pub2)
	require.Error(t, err)
}
