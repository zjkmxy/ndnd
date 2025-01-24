package object

import (
	"fmt"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	sec "github.com/named-data/ndnd/std/security"
	"github.com/named-data/ndnd/std/security/signer"
)

// SuggestSigner returns the signer for a given name
// nil is returned if no signer is found
func (c *Client) SuggestSigner(name enc.Name) ndn.Signer {
	if c.trust == nil {
		return signer.NewSha256Signer()
	}
	name = removeSegVer(name)
	return c.trust.Suggest(name)
}

// Validate a data packet using the client configuration
func (c *Client) Validate(data ndn.Data, sigCov enc.Wire, callback func(bool, error)) {
	c.ValidateExt(ndn.ValidateExtArgs{
		Data:       data,
		SigCovered: sigCov,
		Callback:   callback,
	})
}

// ValidateExt is an advanced API for validating data packets
func (c *Client) ValidateExt(args ndn.ValidateExtArgs) {
	if c.trust == nil {
		args.Callback(true, nil)
		return
	}

	// Pop off the version and segment components
	overrideName := removeSegVer(args.Data.Name())
	if len(args.OverrideName) > 0 {
		overrideName = args.OverrideName
	}

	// Add to queue of validation
	select {
	case c.validatepipe <- sec.TrustConfigValidateArgs{
		Data:         args.Data,
		DataSigCov:   args.SigCovered,
		Callback:     args.Callback,
		OverrideName: overrideName,
		Fetch: func(name enc.Name, config *ndn.InterestConfig, callback ndn.ExpressCallbackFunc) {
			// Pass through extra options
			if args.CertNextHop != nil {
				config.NextHopId = args.CertNextHop
			}

			// Express the interest with reliability
			c.ExpressR(ndn.ExpressRArgs{
				Name:     name,
				Config:   config,
				Retries:  3,
				Callback: callback,
			})
		},
	}:
		// Queued successfully
	default:
		args.Callback(false, fmt.Errorf("validation queue full"))
	}
}

// removeSegVer removes the segment and version components from a name
func removeSegVer(name enc.Name) enc.Name {
	if len(name) > 2 {
		if name[len(name)-1].Typ == enc.TypeSegmentNameComponent {
			name = name[:len(name)-1]
		}
		if name[len(name)-1].Typ == enc.TypeVersionNameComponent {
			name = name[:len(name)-1]
		}
	}
	return name
}
