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

// Name: /ndn/alice/KEY/cK%1D%A4%E1%5B%91%CF
// SigType: Ed25519
const KEY_ALICE = `
BsoHGwgDbmRuCAVhbGljZQgDS0VZCAhjSx2k4VuRzxQDGAEJFUC64F62YK0/v5z4
fjONZO7Y4PNqy7FiDnar33uVO71FLK6Vp8GrPCkEhuODl6GBv2nUuovtO9KtHW11
8apSS093FiIbAQUcHQcbCANuZG4IBWFsaWNlCANLRVkICGNLHaThW5HPF0Cw3Oh7
I2jmBBxop1bIPXq292TfltVwhdbB3/yUXkKcg3BYbY6vcAhNNqrG2B+G/iHvKGsy
DpvDtnlEN72hIeIP
`

// Name: /ndn/KEY/%27%C4%B2%2A%9F%7B%81%27/ndn/v=1651246789556
// SigType: ECDSA-SHA256
// Validity: 2022-04-29 15:39:50 +0000 UTC - 2026-12-31 23:59:59 +0000 UTC
const CERT_ROOT = `
Bv0BSwcjCANuZG4IA0tFWQgIJ8SyKp97gScIA25kbjYIAAABgHX6c7QUCRgBAhkE
ADbugBVbMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEPuDnW4oq0mULLT8PDXh0
zuBg+0SJ1yPC85jylUU+hgxX9fDNyjlykLrvb1D6IQRJWJHMKWe6TJKPUhGgOT65
8hZyGwEDHBYHFAgDbmRuCANLRVkICCfEsiqfe4En/QD9Jv0A/g8yMDIyMDQyOVQx
NTM5NTD9AP8PMjAyNjEyMzFUMjM1OTU5/QECKf0CACX9AgEIZnVsbG5hbWX9AgIV
TkROIFRlc3RiZWQgUm9vdCAyMjA0F0gwRgIhAPYUOjNakdfDGh5j9dcCGOz+Ie1M
qoAEsjM9PEUEWbnqAiEApu0rg9GAK1LNExjLYAF6qVgpWQgU+atPn63Gtuubqyg=
`

var ISSUER = enc.NewStringComponent(enc.TypeGenericNameComponent, "myissuer")
var KEY_ALICE_NAME, _ = enc.NameFromStr("/ndn/alice/KEY/cK%1D%A4%E1%5B%91%CF")
var KEY_ROOT_NAME, _ = enc.NameFromStr("/ndn/KEY/%27%C4%B2%2A%9F%7B%81%27")

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
	aliceSigner := utils.WithoutErr(signer.DecodeSecret(aliceKeyData))

	// self-sign alice's key
	aliceCert, err := sec.SignCert(sec.SignCertArgs{
		Signer:   aliceSigner,
		Data:     aliceKeyData,
		IssuerId: ISSUER,
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

	// check data attributes
	require.Equal(t, ndn.ContentTypeKey, *cert.ContentType())

	// check data content is public key
	alicePub := utils.WithoutErr(aliceSigner.Public())
	require.Equal(t, alicePub, cert.Content().Join())

	// check signature format
	signature := cert.Signature()
	require.Equal(t, ndn.SignatureEd25519, signature.SigType())
	require.Equal(t, aliceSigner.KeyName(), signature.KeyName())

	// check signature
	require.Equal(t, 64, len(signature.SigValue())) // ed25519
	require.True(t, signer.Ed25519Validate(certSigCovered, signature, alicePub))
}

func TestSignCertOther(t *testing.T) {
	utils.SetTestingT(t)

	aliceKey, _ := base64.StdEncoding.DecodeString(KEY_ALICE)
	aliceKeyData, _, _ := spec_2022.Spec{}.ReadData(enc.NewBufferReader(aliceKey))
	aliceSigner := utils.WithoutErr(signer.DecodeSecret(aliceKeyData))

	// parse existing certificate
	rootCert, _ := base64.StdEncoding.DecodeString(CERT_ROOT)
	rootCertData, _, _ := spec_2022.Spec{}.ReadData(enc.NewBufferReader(rootCert))

	// sign root cert with alice's key
	newCertB, err := sec.SignCert(sec.SignCertArgs{
		Signer:   aliceSigner,
		Data:     rootCertData,
		IssuerId: ISSUER,
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

	// check data attributes
	require.Equal(t, ndn.ContentTypeKey, *newCert.ContentType())

	// check data content is public key
	rootPub := rootCertData.Content().Join()
	require.Equal(t, rootPub, newCert.Content().Join())

	// check signature format
	signature := newCert.Signature()
	require.Equal(t, ndn.SignatureEd25519, signature.SigType())
	require.Equal(t, aliceSigner.KeyName(), signature.KeyName())

	// check signature
	alicePub := utils.WithoutErr(aliceSigner.Public())
	require.Equal(t, 64, len(signature.SigValue())) // ed25519
	require.True(t, signer.Ed25519Validate(newSigCov, signature, alicePub))
}
