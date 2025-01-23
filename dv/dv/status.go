package dv

import (
	"time"

	"github.com/named-data/ndnd/dv/tlv"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	sig "github.com/named-data/ndnd/std/security/signer"
	"github.com/named-data/ndnd/std/utils"
)

// Received advertisement Interest
func (dv *Router) statusOnInterest(args ndn.InterestHandlerArgs) {
	status := func() tlv.Status {
		dv.mutex.Lock()
		defer dv.mutex.Unlock()
		return tlv.Status{
			Version:     utils.NDNdVersion,
			NetworkName: &tlv.Destination{Name: dv.config.NetworkName()},
			RouterName:  &tlv.Destination{Name: dv.config.RouterName()},
			NRibEntries: uint64(dv.rib.Size()),
			NNeighbors:  uint64(dv.neighbors.Size()),
			NFibEntries: uint64(dv.fib.Size()),
		}
	}()

	name := args.Interest.Name()
	cfg := &ndn.DataConfig{
		ContentType: utils.IdPtr(ndn.ContentTypeBlob),
		Freshness:   utils.IdPtr(time.Second),
	}
	signer := sig.NewSha256Signer()

	data, err := dv.engine.Spec().MakeData(name, cfg, status.Encode(), signer)
	if err != nil {
		log.Warn(dv, "Failed to make readvertise response Data", "err", err)
		return
	}

	args.Reply(data.Wire)
}
