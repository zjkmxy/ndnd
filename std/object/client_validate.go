package object

import (
	"fmt"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	sec "github.com/named-data/ndnd/std/security"
)

// Validate a data packet using the client configuration
func (c *Client) Validate(data ndn.Data, callback func(bool, error)) {
	if c.trust == nil {
		callback(true, nil)
		return
	}

	// Pop off the version and segment components
	name := data.Name()
	if len(name) > 2 {
		if name[len(name)-1].Typ == enc.TypeSegmentNameComponent {
			name = name[:len(name)-1]
		}
		if name[len(name)-1].Typ == enc.TypeVersionNameComponent {
			name = name[:len(name)-1]
		}
	}

	// Add to queue of validation
	select {
	case c.validatepipe <- sec.ValidateArgs{
		Data:     data,
		Callback: callback,
		DataName: name,
		Fetch: func(name enc.Name, config *ndn.InterestConfig, found func(ndn.Data, []byte, error)) {
			c.ExpressR(ndn.ExpressRArgs{
				Name:    name,
				Config:  config,
				Retries: 3,
				Callback: func(res ndn.ExpressCallbackArgs) {
					if res.Result == ndn.InterestResultData {
						found(res.Data, res.RawData.Join(), nil)
					} else if res.Error != nil {
						found(nil, nil, res.Error)
					} else {
						found(nil, nil, fmt.Errorf("failed to fetch certificate (%s) with result: %v", name, res.Result))
					}
				},
			})
		},
	}:
		// Queued successfully
	default:
		callback(false, fmt.Errorf("validation queue full"))
	}
}
