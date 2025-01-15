package security

import (
	"encoding/pem"
	"errors"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"

	spec "github.com/named-data/ndnd/std/ndn/spec_2022"
)

const TXT_TYPE_CERT = "NDN Certificate"
const TXT_TYPE_SECRET = "NDN Key"

// TxtFrom converts an NDN data to a text representation following RFC 7468.
func TxtFrom(raw []byte) ([]byte, error) {
	data, _, err := spec.Spec{}.ReadData(enc.NewBufferReader(raw))
	if err != nil {
		return nil, err
	}

	if data.ContentType() == nil {
		return nil, errors.New("missing content type")
	}

	// Explanatory text before the block
	headers := map[string]string{
		"Name": data.Name().String(),
	}

	var pemType string
	switch *data.ContentType() {
	case ndn.ContentTypeKey:
		pemType = TXT_TYPE_CERT
	case ndn.ContentTypeSecret:
		pemType = TXT_TYPE_SECRET
	default:
		return nil, errors.New("unsupported content type")
	}

	if nb, na := data.Signature().Validity(); nb != nil && na != nil {
		headers["Validity"] = nb.String() + " - " + na.String()
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:    pemType,
		Headers: headers,
		Bytes:   raw,
	}), nil
}

// TxtParse converts a text representation of an NDN data.
func TxtParse(txt []byte) [][]byte {
	ret := make([][]byte, 0)

	for {
		block, rest := pem.Decode(txt)
		if block == nil {
			break
		}
		ret = append(ret, block.Bytes)
		txt = rest
	}

	return ret
}
