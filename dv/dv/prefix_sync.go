package dv

import (
	"time"

	"github.com/named-data/ndnd/dv/table"
	"github.com/named-data/ndnd/dv/tlv"
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	ndn_sync "github.com/named-data/ndnd/std/sync"
	"github.com/named-data/ndnd/std/utils"
)

// Fetch all required prefix data
func (dv *Router) prefixDataFetchAll() {
	dv.mutex.Lock()
	defer dv.mutex.Unlock()

	for _, e := range dv.rib.Entries() {
		router := dv.pfx.GetRouter(e.Name())
		if router.Known < router.Latest {
			go dv.prefixDataFetch(e.Name())
		}
	}
}

// Received prefix sync update
func (dv *Router) onPfxSyncUpdate(ssu ndn_sync.SvSyncUpdate) {
	dv.mutex.Lock()
	defer dv.mutex.Unlock()

	// Update the prefix table
	dv.pfx.GetRouter(ssu.NodeId).Latest = ssu.High

	// Start a fetching thread (if needed)
	go dv.prefixDataFetch(ssu.NodeId)
}

// Fetch prefix data
func (dv *Router) prefixDataFetch(nodeId enc.Name) {
	dv.mutex.Lock()
	defer dv.mutex.Unlock()

	// Check if the RIB has this destination
	if !dv.rib.Has(nodeId) {
		return
	}

	// At any given time, there is only one thread fetching
	// prefix data for a node. This thread recursively calls itself.
	router := dv.pfx.GetRouter(nodeId)
	if router.Fetching || router.Known >= router.Latest {
		return
	}

	// Mark this node as fetching
	router.Fetching = true

	// Fetch the prefix data
	log.Debugf("prefixDataFetch: fetching prefix data for %s [%d => %d]", nodeId, router.Known, router.Latest)

	cfg := &ndn.InterestConfig{
		MustBeFresh: true,
		Lifetime:    utils.IdPtr(4 * time.Second),
		Nonce:       utils.ConvertNonce(dv.engine.Timer().Nonce()),
	}

	isSnap := router.Latest-router.Known > 100
	name := append(nodeId,
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "DV"),
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "PFX"),
	)
	if isSnap {
		name = append(name, enc.NewStringComponent(enc.TypeKeywordNameComponent, "SNAP"))
		cfg.CanBePrefix = true
	} else {
		name = append(name, enc.NewSequenceNumComponent(router.Known+1))
	}

	interest, err := dv.engine.Spec().MakeInterest(name, cfg, nil, nil)
	if err != nil {
		log.Warnf("prefixDataFetch: failed to make Interest: %+v", err)
		return
	}

	err = dv.engine.Express(interest, func(args ndn.ExpressCallbackArgs) {
		go func() {
			// Done fetching, restart if needed
			defer func() {
				dv.mutex.Lock()
				defer dv.mutex.Unlock()

				router.Fetching = false
				go dv.prefixDataFetch(nodeId) // recheck
			}()

			// Sleep this goroutine if no data was received
			if args.Result != ndn.InterestResultData {
				log.Warnf("prefixDataFetch: failed to fetch prefix data %s: %d", interest.FinalName, args.Result)

				// see advertDataFetch
				if args.Result != ndn.InterestResultTimeout {
					time.Sleep(2 * time.Second)
				} else {
					time.Sleep(100 * time.Millisecond)
				}
				return
			}

			dv.processPrefixData(args.Data, router)
		}()
	})
	if err != nil {
		log.Warnf("prefixDataFetch: failed to express Interest: %+v", err)
		return
	}
}

func (dv *Router) processPrefixData(data ndn.Data, router *table.PrefixTableRouter) {
	ops, err := tlv.ParsePrefixOpList(enc.NewWireReader(data.Content()), true)
	if err != nil {
		log.Warnf("prefixDataFetch: failed to parse PrefixOpList: %+v", err)
		return
	}

	dv.mutex.Lock()
	defer dv.mutex.Unlock()

	// Update sequence number
	dataName := data.Name()
	seqNo := dataName[len(dataName)-1]
	if seqNo.Typ != enc.TypeSequenceNumNameComponent {
		log.Warnf("prefixDataFetch: unexpected sequence number type: %s", seqNo.Typ)
		return
	}

	// Update the prefix table
	router.Known = seqNo.NumberVal()
	if dv.pfx.Apply(ops) {
		// Update the local fib if prefix table changed (very expensive)
		go dv.fibUpdate()
	}
}
