package keychain_test

import (
	"encoding/base64"
	"os"
	"testing"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/object"
	sec "github.com/named-data/ndnd/std/security"
	"github.com/named-data/ndnd/std/security/keychain"
	sig "github.com/named-data/ndnd/std/security/signer"
	"github.com/named-data/ndnd/std/utils"
	"github.com/stretchr/testify/require"
)

const CERT_ROOT = `
Bv0BSwcjCANuZG4IA0tFWQgIJ8SyKp97gScIA25kbjYIAAABgHX6c7QUCRgBAhkE
ADbugBVbMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEPuDnW4oq0mULLT8PDXh0
zuBg+0SJ1yPC85jylUU+hgxX9fDNyjlykLrvb1D6IQRJWJHMKWe6TJKPUhGgOT65
8hZyGwEDHBYHFAgDbmRuCANLRVkICCfEsiqfe4En/QD9Jv0A/g8yMDIyMDQyOVQx
NTM5NTD9AP8PMjAyNjEyMzFUMjM1OTU5/QECKf0CACX9AgEIZnVsbG5hbWX9AgIV
TkROIFRlc3RiZWQgUm9vdCAyMjA0F0gwRgIhAPYUOjNakdfDGh5j9dcCGOz+Ie1M
qoAEsjM9PEUEWbnqAiEApu0rg9GAK1LNExjLYAF6qVgpWQgU+atPn63Gtuubqyg=
`

const KEY_ALICE = `
-----BEGIN NDN Key-----
Name: /ndn/alice/KEY/cK%1D%A4%E1%5B%91%CF
SigType: Ed25519

BsoHGwgDbmRuCAVhbGljZQgDS0VZCAhjSx2k4VuRzxQDGAEJFUC64F62YK0/v5z4
fjONZO7Y4PNqy7FiDnar33uVO71FLK6Vp8GrPCkEhuODl6GBv2nUuovtO9KtHW11
8apSS093FiIbAQUcHQcbCANuZG4IBWFsaWNlCANLRVkICGNLHaThW5HPF0Cw3Oh7
I2jmBBxop1bIPXq292TfltVwhdbB3/yUXkKcg3BYbY6vcAhNNqrG2B+G/iHvKGsy
DpvDtnlEN72hIeIP
-----END NDN Key-----
`

var CERT_ROOT_NAME, _ = enc.NameFromStr("/ndn/KEY/%27%C4%B2%2A%9F%7B%81%27/ndn/v=1651246789556")
var KEY_ALICE_NAME, _ = enc.NameFromStr("/ndn/alice/KEY/cK%1D%A4%E1%5B%91%CF")

func TestKeyChainMem(t *testing.T) {
	utils.SetTestingT(t)

	store := object.NewMemoryStore()
	kc := keychain.NewKeyChainMem(store)

	// Insert a key
	idName, _ := enc.NameFromStr("/my/test/identity")
	signer := utils.WithoutErr(sig.KeygenEd25519(sec.MakeKeyName(idName)))
	require.NoError(t, kc.InsertKey(signer))

	// Check key in keychain
	identity := kc.GetIdentity(idName)
	require.NotNil(t, identity)
	require.Equal(t, idName, identity.Name())
	require.Len(t, identity.AllSigners(), 1)
	require.Equal(t, signer, identity.Signer())

	// Insert another key for the same identity
	signer2 := utils.WithoutErr(sig.KeygenEd25519(sec.MakeKeyName(idName)))
	require.NoError(t, kc.InsertKey(signer2))
	identity = kc.GetIdentity(idName)
	require.NotNil(t, identity)
	require.Len(t, identity.AllSigners(), 2)
	require.Equal(t, signer2, identity.Signer())

	// Lookup non-existing identity
	idName2, _ := enc.NameFromStr("/my/test/identity2")
	identity = kc.GetIdentity(idName2)
	require.Nil(t, identity)

	// Insert key for different identity
	signer3 := utils.WithoutErr(sig.KeygenEd25519(sec.MakeKeyName(idName2)))
	require.NoError(t, kc.InsertKey(signer3))
	identity = kc.GetIdentity(idName2)
	require.NotNil(t, identity)
	require.Len(t, identity.AllSigners(), 1)
	require.Equal(t, signer3, identity.Signer())

	// Insert invalid key
	signer4 := sig.NewSha256Signer()
	require.Error(t, kc.InsertKey(signer4))

	// Insert a certificate.
	certRoot, _ := base64.StdEncoding.DecodeString(CERT_ROOT)
	require.NoError(t, kc.InsertCert(certRoot))

	// Check certificate in store
	data, err := store.Get(CERT_ROOT_NAME, false)
	require.NoError(t, err)
	require.Equal(t, certRoot, data)
}

func TestKeyChainDir(t *testing.T) {
	utils.SetTestingT(t)

	store := object.NewMemoryStore()

	// Create a temporary directory
	dirname := "./ndn-test-keychain"
	require.NoError(t, os.RemoveAll(dirname))
	defer os.RemoveAll(dirname)
	require.NoError(t, os.Mkdir(dirname, 0755))

	// Write root cert (raw)
	rootCert, _ := base64.StdEncoding.DecodeString(CERT_ROOT)
	require.NoError(t, os.WriteFile(dirname+"/root.cert", rootCert, 0644))

	// Write Alice key (text)
	require.NoError(t, os.WriteFile(dirname+"/alice.key", []byte(KEY_ALICE), 0644))

	// Create a keychain
	kc, err := keychain.NewKeyChainDir(dirname, store)
	require.NoError(t, err)

	// Check root cert
	data, err := store.Get(CERT_ROOT_NAME, false)
	require.NoError(t, err)
	require.Equal(t, rootCert, data)

	// Check Alice key
	identity := kc.GetIdentity(KEY_ALICE_NAME[:2])
	require.NotNil(t, identity)
	require.Len(t, identity.AllSigners(), 1)
	require.Equal(t, identity.Signer().KeyName(), KEY_ALICE_NAME)

	// Check Alice key is not in store
	data, _ = store.Get(KEY_ALICE_NAME, false)
	require.Nil(t, data)
}
