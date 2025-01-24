package security_test

import (
	"encoding/base64"
	"testing"

	"github.com/named-data/ndnd/std/security"
	"github.com/named-data/ndnd/std/utils"
	"github.com/stretchr/testify/require"
)

func TestPemEncodeCert(t *testing.T) {
	utils.SetTestingT(t)

	cert, _ := base64.StdEncoding.DecodeString(CERT_ROOT)
	res := utils.WithoutErr(security.PemEncode(cert))
	require.Equal(t, CERT_ROOT_PEM, string(res))
}

func TestPemDecodeCert(t *testing.T) {
	utils.SetTestingT(t)

	cert, _ := base64.StdEncoding.DecodeString(CERT_ROOT)
	res := security.PemDecode([]byte(CERT_ROOT_PEM))
	require.Equal(t, cert, res[0])
}
