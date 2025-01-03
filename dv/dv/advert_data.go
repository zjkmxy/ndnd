package dv

import (
	"time"

	"github.com/named-data/ndnd/dv/tlv"
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/security"
	"github.com/named-data/ndnd/std/utils"
)

func (dv *Router) advertDataFetch(nodeId enc.Name, seqNo uint64) {
	// debounce; wait before fetching, then check if this is still the latest
	// sequence number known for this neighbor
	time.Sleep(10 * time.Millisecond)
	if ns := dv.neighbors.Get(nodeId); ns == nil || ns.AdvertSeq != seqNo {
		return
	}

	advName := append(enc.Name{enc.LOCALHOP}, append(nodeId,
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "DV"),
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "ADV"),
		enc.NewSequenceNumComponent(seqNo), // unused for now
	)...)

	// Fetch the advertisement
	cfg := &ndn.InterestConfig{
		MustBeFresh: true,
		Lifetime:    utils.IdPtr(4 * time.Second),
		Nonce:       utils.ConvertNonce(dv.engine.Timer().Nonce()),
	}

	interest, err := dv.engine.Spec().MakeInterest(advName, cfg, nil, nil)
	if err != nil {
		log.Warnf("advertDataFetch: failed to make Interest: %+v", err)
		return
	}

	// Fetch the advertisement
	err = dv.engine.Express(interest, func(args ndn.ExpressCallbackArgs) {
		go func() { // Don't block the main loop
			if args.Result != ndn.InterestResultData {
				// If this wasn't a timeout, wait for 2s before retrying
				// This prevents excessive retries in case of NACKs
				if args.Result != ndn.InterestResultTimeout {
					time.Sleep(2 * time.Second)
				} else {
					time.Sleep(100 * time.Millisecond)
				}

				// Keep retrying until we get the advertisement
				// If the router is dead, we break out of this by checking
				// that the sequence number is gone (above)
				log.Warnf("advertDataFetch: retrying %s: %+v", interest.FinalName.String(), args.Result)
				dv.advertDataFetch(nodeId, seqNo)
				return
			}

			// Process the advertisement
			dv.advertDataHandler(args.Data)
		}()
	})
	if err != nil {
		log.Warnf("advertDataFetch: failed to express Interest: %+v", err)
	}
}

// Received advertisement Interest
func (dv *Router) advertDataOnInterest(args ndn.InterestHandlerArgs) {
	// For now, just send the latest advertisement at all times
	// This will need to change if we switch to differential updates

	// TODO: sign the advertisement
	signer := security.NewSha256Signer()

	// Encode latest advertisement
	content := func() *tlv.Advertisement {
		dv.mutex.Lock()
		defer dv.mutex.Unlock()
		return dv.rib.Advert()
	}().Encode()

	data, err := dv.engine.Spec().MakeData(
		args.Interest.Name(),
		&ndn.DataConfig{
			ContentType: utils.IdPtr(ndn.ContentTypeBlob),
			Freshness:   utils.IdPtr(10 * time.Second),
		},
		content,
		signer)
	if err != nil {
		log.Warnf("advertDataOnInterest: failed to make Data: %+v", err)
		return
	}

	// Send the Data packet
	err = args.Reply(data.Wire)
	if err != nil {
		log.Warnf("advertDataOnInterest: failed to reply: %+v", err)
		return
	}
}

// Received advertisement Data
func (dv *Router) advertDataHandler(data ndn.Data) {
	// Parse name components
	name := data.Name()
	neighbor := name[1 : len(name)-3]
	seqNo := name[len(name)-1].NumberVal()

	// Lock DV state
	dv.mutex.Lock()
	defer dv.mutex.Unlock()

	// Check if this is the latest advertisement
	ns := dv.neighbors.Get(neighbor)
	if ns == nil {
		log.Warnf("advertDataHandler: unknown advertisement %s", neighbor)
		return
	}
	if ns.AdvertSeq != seqNo {
		log.Debugf("advertDataHandler: old advertisement for %s (%d != %d)", neighbor, ns.AdvertSeq, seqNo)
		return
	}

	// TODO: verify signature on Advertisement
	log.Debugf("advertDataHandler: received: %s", data.Name())

	// Parse the advertisement
	raw := data.Content().Join() // clone
	advert, err := tlv.ParseAdvertisement(enc.NewBufferReader(raw), false)
	if err != nil {
		log.Errorf("advertDataHandler: failed to parse advertisement: %+v", err)
		return
	}

	// Update the local advertisement list
	ns.Advert = advert
	go dv.ribUpdate(ns)
}
