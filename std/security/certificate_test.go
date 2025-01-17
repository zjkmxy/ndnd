package security_test

import (
	"encoding/base64"
	"testing"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/ndn/spec_2022"
	sec "github.com/named-data/ndnd/std/security"
	"github.com/named-data/ndnd/std/security/signer"
	"github.com/named-data/ndnd/std/utils"
	"github.com/stretchr/testify/require"
)

func TestSignCertInvalid(t *testing.T) {
	utils.SetTestingT(t)

	_, err := sec.SignCert(sec.SignCertArgs{
		Signer:   nil,
		Data:     nil,
		IssuerId: ISSUER,
	})
	require.Error(t, err)
}

func TestSignCertSelf(t *testing.T) {
	utils.SetTestingT(t)

	aliceKey, _ := base64.StdEncoding.DecodeString(KEY_ALICE)
	aliceKeyData, _, _ := spec_2022.Spec{}.ReadData(enc.NewBufferReader(aliceKey))
	aliceSigner := utils.WithoutErr(signer.UnmarshalSecret(aliceKeyData))

	// self-sign alice's key
	aliceCert, err := sec.SignCert(sec.SignCertArgs{
		Signer:    aliceSigner,
		Data:      aliceKeyData,
		IssuerId:  ISSUER,
		NotBefore: T1,
		NotAfter:  T2,
	})
	require.NoError(t, err)
	cert, certSigCovered, err := spec_2022.Spec{}.ReadData(enc.NewWireReader(aliceCert))
	require.NoError(t, err)

	// check certificate name
	name := cert.Name()
	require.True(t, KEY_ALICE_NAME.IsPrefix(name))
	require.Equal(t, len(KEY_ALICE_NAME)+2, len(name))
	require.Equal(t, enc.TypeVersionNameComponent, name[len(name)-1].Typ)
	require.Greater(t, name[len(name)-1].NumberVal(), uint64(0))
	require.Equal(t, ISSUER, name[len(name)-2])

	// check data content is public key
	require.Equal(t, ndn.ContentTypeKey, *cert.ContentType())
	alicePub := utils.WithoutErr(aliceSigner.Public())
	require.Equal(t, alicePub, cert.Content().Join())

	// check signature format
	signature := cert.Signature()
	require.Equal(t, ndn.SignatureEd25519, signature.SigType())
	require.Equal(t, aliceSigner.KeyName(), signature.KeyName())

	// check validity period
	notBefore, notAfter := signature.Validity()
	require.Equal(t, T1, *notBefore)
	require.Equal(t, T2, *notAfter)

	// check signature
	require.Equal(t, 64, len(signature.SigValue())) // ed25519
	require.True(t, signer.Ed25519Validate(certSigCovered, signature, alicePub))
}

func TestSignCertOther(t *testing.T) {
	utils.SetTestingT(t)

	aliceKey, _ := base64.StdEncoding.DecodeString(KEY_ALICE)
	aliceKeyData, _, _ := spec_2022.Spec{}.ReadData(enc.NewBufferReader(aliceKey))
	aliceSigner := utils.WithoutErr(signer.UnmarshalSecret(aliceKeyData))

	// parse existing certificate
	rootCert, _ := base64.StdEncoding.DecodeString(CERT_ROOT)
	rootCertData, _, _ := spec_2022.Spec{}.ReadData(enc.NewBufferReader(rootCert))

	// sign root cert with alice's key
	newCertB, err := sec.SignCert(sec.SignCertArgs{
		Signer:    aliceSigner,
		Data:      rootCertData,
		IssuerId:  ISSUER,
		NotBefore: T1,
		NotAfter:  T2,
	})
	require.NoError(t, err)
	newCert, newSigCov, err := spec_2022.Spec{}.ReadData(enc.NewWireReader(newCertB))
	require.NoError(t, err)

	// check certificate name
	name := newCert.Name()
	require.True(t, KEY_ROOT_NAME.IsPrefix(name))
	require.Equal(t, len(KEY_ROOT_NAME)+2, len(name))
	require.Equal(t, enc.TypeVersionNameComponent, name[len(name)-1].Typ)
	require.Greater(t, name[len(name)-1].NumberVal(), uint64(0))
	require.Equal(t, ISSUER, name[len(name)-2])

	// check data content is public key
	require.Equal(t, ndn.ContentTypeKey, *newCert.ContentType())
	rootPub := rootCertData.Content().Join()
	require.Equal(t, rootPub, newCert.Content().Join())

	// check signature format
	signature := newCert.Signature()
	require.Equal(t, ndn.SignatureEd25519, signature.SigType())
	require.Equal(t, aliceSigner.KeyName(), signature.KeyName())

	// check validity period
	notBefore, notAfter := signature.Validity()
	require.Equal(t, T1, *notBefore)
	require.Equal(t, T2, *notAfter)

	// check signature
	alicePub := utils.WithoutErr(aliceSigner.Public())
	require.Equal(t, 64, len(signature.SigValue())) // ed25519
	require.True(t, signer.Ed25519Validate(newSigCov, signature, alicePub))
}
