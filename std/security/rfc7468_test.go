package security_test

import (
	"encoding/base64"
	"testing"

	"github.com/named-data/ndnd/std/security"
	"github.com/named-data/ndnd/std/utils"
	"github.com/stretchr/testify/require"
)

const CERT_B64 = `
Bv0BSwcjCANuZG4IA0tFWQgIJ8SyKp97gScIA25kbjYIAAABgHX6c7QUCRgBAhkE
ADbugBVbMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEPuDnW4oq0mULLT8PDXh0
zuBg+0SJ1yPC85jylUU+hgxX9fDNyjlykLrvb1D6IQRJWJHMKWe6TJKPUhGgOT65
8hZyGwEDHBYHFAgDbmRuCANLRVkICCfEsiqfe4En/QD9Jv0A/g8yMDIyMDQyOVQx
NTM5NTD9AP8PMjAyNjEyMzFUMjM1OTU5/QECKf0CACX9AgEIZnVsbG5hbWX9AgIV
TkROIFRlc3RiZWQgUm9vdCAyMjA0F0gwRgIhAPYUOjNakdfDGh5j9dcCGOz+Ie1M
qoAEsjM9PEUEWbnqAiEApu0rg9GAK1LNExjLYAF6qVgpWQgU+atPn63Gtuubqyg=`

const CERT_PEM = `-----BEGIN NDN Certificate-----
Key: /ndn/KEY/%27%C4%B2%2A%9F%7B%81%27
Name: /ndn/KEY/%27%C4%B2%2A%9F%7B%81%27/ndn/v=1651246789556
SigType: ECDSA-SHA256
Validity: 2022-04-29 15:39:50 +0000 UTC - 2026-12-31 23:59:59 +0000 UTC

Bv0BSwcjCANuZG4IA0tFWQgIJ8SyKp97gScIA25kbjYIAAABgHX6c7QUCRgBAhkE
ADbugBVbMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEPuDnW4oq0mULLT8PDXh0
zuBg+0SJ1yPC85jylUU+hgxX9fDNyjlykLrvb1D6IQRJWJHMKWe6TJKPUhGgOT65
8hZyGwEDHBYHFAgDbmRuCANLRVkICCfEsiqfe4En/QD9Jv0A/g8yMDIyMDQyOVQx
NTM5NTD9AP8PMjAyNjEyMzFUMjM1OTU5/QECKf0CACX9AgEIZnVsbG5hbWX9AgIV
TkROIFRlc3RiZWQgUm9vdCAyMjA0F0gwRgIhAPYUOjNakdfDGh5j9dcCGOz+Ie1M
qoAEsjM9PEUEWbnqAiEApu0rg9GAK1LNExjLYAF6qVgpWQgU+atPn63Gtuubqyg=
-----END NDN Certificate-----
`

func TestPemEncodeCert(t *testing.T) {
	utils.SetTestingT(t)

	cert, _ := base64.StdEncoding.DecodeString(CERT_B64)
	res := utils.WithoutErr(security.PemEncode(cert))
	require.Equal(t, CERT_PEM, string(res))
}

func TestPemDecodeCert(t *testing.T) {
	utils.SetTestingT(t)

	cert, _ := base64.StdEncoding.DecodeString(CERT_B64)
	res := security.PemDecode([]byte(CERT_PEM))
	require.Equal(t, cert, res[0])
}
