package object

import (
	"fmt"

	enc "github.com/named-data/ndnd/std/encoding"
	rdr "github.com/named-data/ndnd/std/ndn/rdr_2024"
	spec "github.com/named-data/ndnd/std/ndn/spec_2022"
)

// LatestLocal returns the latest version name of an object in the store
func (c *Client) LatestLocal(name enc.Name) (enc.Name, error) {
	raw, err := c.store.Get(name.Append(enc.NewKeywordComponent(rdr.MetadataKeyword)), true)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	data, _, err := spec.Spec{}.ReadData(enc.NewBufferView(raw))
	if err != nil {
		return nil, err
	}

	version := data.Name().At(-2)
	if !version.IsVersion() {
		return nil, fmt.Errorf("invalid metadata for %s", name)
	}

	return name.Append(version), nil
}
