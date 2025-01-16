package keychain_test

import (
	"encoding/base64"
	"testing"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	spec "github.com/named-data/ndnd/std/ndn/spec_2022"
	"github.com/named-data/ndnd/std/security/crypto"
	"github.com/named-data/ndnd/std/security/keychain"
	"github.com/named-data/ndnd/std/utils"
	"github.com/stretchr/testify/require"
)

const RSA_KEY_SECRET = `
MIICXQIBAAKBgQDgqaPyTpqeBqI+o4QhZUHBl9gSE25Jv+1TDSSECWmwmxbAdisPL1AmWaiBcm
MA2/tdFIfbsOfOVD27BVN9/WzPszc2I4njUrSz4d5L2e83veK+Q9UaG/wHqe7fG6Hj/RHzt5aj
fNjJgBKH6mUUgJWQ2QImPYrEmgk3t7KXrLq1EQIDAQABAoGAKBSbtyLm2sJ8N4icjgiujoc0eS
UWS/n9sQ9rMFMtk+BXUsbCL9dVCUJ9mXp6xzB3y8dZ5YvODzVgPflZR+TqgF3R4ii4AVLdSsJ5
9QOVLh9d/vkij3v41/wgPaFXqIweW4S5t1uXvlC+O7B7MADDR/VzhR8KarTTCCMk9xjCowECQQ
D+6Vo50oq3z+qYVQUItWfmy2NoLt8082k6+SVHnrFT622tXqtSllB4VwUPhzRqOXkSJvrfJJBF
7ZTkyoFKpRapAkEA4Z849HD5QPpiHyGXcibxz0s9RkorK4PYvccuihDRtpMLcjz1ryPdeanl2Z
PbzATYnVqhoowtKdsRTcJKdMn0KQJBALoA8mpQ3BHGMCtZlmPFYvyAmpex5ANSPg3fMLmy7TgM
CSrBcoe/0RYOgU3UXYXJTDPXp6Vdm7y64LOVpIQgNIkCQQCCkQf+vZog9kTuSxw/XTY2hg4RrT
5KUmSNfsT59T3HcFUBaTGshw7WJ3HyddSOvoc0mIxNat2ACVx8KWG5MF3xAkBIJTjV88JiLxzX
cUQ+mKiK0IHYphhsp+OJqYOA+DRtBfL7NRVXV1vjgMTA/h6mtCZkTz+3DLu+Hbe+p0i/rwen
`

const RSA_KEY_DATA = `
Bv0DCQcJCAJteQgDa2V5FAMYAQkV/QJhMIICXQIBAAKBgQDgqaPyTpqeBqI+o4Qh
ZUHBl9gSE25Jv+1TDSSECWmwmxbAdisPL1AmWaiBcmMA2/tdFIfbsOfOVD27BVN9
/WzPszc2I4njUrSz4d5L2e83veK+Q9UaG/wHqe7fG6Hj/RHzt5ajfNjJgBKH6mUU
gJWQ2QImPYrEmgk3t7KXrLq1EQIDAQABAoGAKBSbtyLm2sJ8N4icjgiujoc0eSUW
S/n9sQ9rMFMtk+BXUsbCL9dVCUJ9mXp6xzB3y8dZ5YvODzVgPflZR+TqgF3R4ii4
AVLdSsJ59QOVLh9d/vkij3v41/wgPaFXqIweW4S5t1uXvlC+O7B7MADDR/VzhR8K
arTTCCMk9xjCowECQQD+6Vo50oq3z+qYVQUItWfmy2NoLt8082k6+SVHnrFT622t
XqtSllB4VwUPhzRqOXkSJvrfJJBF7ZTkyoFKpRapAkEA4Z849HD5QPpiHyGXcibx
z0s9RkorK4PYvccuihDRtpMLcjz1ryPdeanl2ZPbzATYnVqhoowtKdsRTcJKdMn0
KQJBALoA8mpQ3BHGMCtZlmPFYvyAmpex5ANSPg3fMLmy7TgMCSrBcoe/0RYOgU3U
XYXJTDPXp6Vdm7y64LOVpIQgNIkCQQCCkQf+vZog9kTuSxw/XTY2hg4RrT5KUmSN
fsT59T3HcFUBaTGshw7WJ3HyddSOvoc0mIxNat2ACVx8KWG5MF3xAkBIJTjV88Ji
LxzXcUQ+mKiK0IHYphhsp+OJqYOA+DRtBfL7NRVXV1vjgMTA/h6mtCZkTz+3DLu+
Hbe+p0i/rwenFhAbAQEcCwcJCAJteQgDa2V5F4Cq9k7cnFIb+sqtEgt84C91Fx0+
BtaAD3KVSpnehHNsYwM90iTWGN+gdBpMHySgWL5BUkBTb9cB+zBLD+/TTpRbDRS6
vaWw6t4JWkwYQT/5U+Ox5bxvi+0nmpQ08FYmpr9qj3B14LvwiSm5bPuhqI50o7zA
OH0CtyGO32zNEtZHxQ==
`

func TestEncodeSecret(t *testing.T) {
	utils.SetTestingT(t)

	// create signer
	name, _ := enc.NameFromStr("/my/key")
	secret, _ := base64.StdEncoding.DecodeString(RSA_KEY_SECRET)
	signer := utils.WithoutErr(crypto.ParseRsa(name, secret))

	// encode signer secret
	wire := utils.WithoutErr(keychain.EncodeSecret(signer))

	// check output data
	data, _, err := spec.Spec{}.ReadData(enc.NewWireReader(wire))
	require.NoError(t, err)
	require.Equal(t, ndn.ContentTypeSecret, *data.ContentType())
	require.Equal(t, name, data.Name())
	require.Equal(t, secret, data.Content().Join())
}

func TestDecodeSecret(t *testing.T) {
	utils.SetTestingT(t)

	// get static secret data
	name, _ := enc.NameFromStr("/my/key")
	dataRaw := utils.WithoutErr(base64.StdEncoding.DecodeString(RSA_KEY_DATA))
	data, _, err := spec.Spec{}.ReadData(enc.NewBufferReader(dataRaw))
	require.NoError(t, err)

	signer, err := keychain.DecodeSecret(data)
	require.NoError(t, err)

	// check output signer
	require.Equal(t, ndn.SignatureSha256WithRsa, signer.Type())
	require.Equal(t, name, signer.KeyName())
}
