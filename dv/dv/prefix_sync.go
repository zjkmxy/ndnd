package dv

import (
	"time"

	"github.com/named-data/ndnd/dv/table"
	"github.com/named-data/ndnd/dv/tlv"
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/object"
	ndn_sync "github.com/named-data/ndnd/std/sync"
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

	// Prefix object for other router
	name := append(nodeId,
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "DV"),
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "PFX"),
	)
	if router.Latest-router.Known > table.PrefixTableSnapThreshold {
		// no version - discover the latest snapshot
		name = append(name, table.PrefixTableSnap)
	} else {
		name = append(name,
			enc.NewSequenceNumComponent(router.Known+1),
			enc.NewVersionComponent(0), // immutable
		)
	}

	dv.client.Consume(name, func(state *object.ConsumeState) bool {
		if !state.IsComplete() {
			return true
		}

		go func() {
			// Done fetching, restart if needed
			defer func() {
				dv.mutex.Lock()
				defer dv.mutex.Unlock()

				router.Fetching = false
				go dv.prefixDataFetch(nodeId) // recheck
			}()

			// Wait before retry if there was a failure
			if err := state.Error(); err != nil {
				log.Warnf("prefixDataFetch: failed to fetch prefix data %s: %+v", name, err)
				time.Sleep(1 * time.Second)
				return
			}

			dv.processPrefixData(state.Name(), state.Content(), router)
		}()

		return true
	})
}

func (dv *Router) processPrefixData(name enc.Name, data []byte, router *table.PrefixTableRouter) {
	ops, err := tlv.ParsePrefixOpList(enc.NewBufferReader(data), true)
	if err != nil {
		log.Warnf("prefixDataFetch: failed to parse PrefixOpList: %+v", err)
		return
	}

	dv.mutex.Lock()
	defer dv.mutex.Unlock()

	// Get sequence number from name
	seqNo := name[len(name)-2]
	if seqNo.Equal(table.PrefixTableSnap) && name[len(name)-1].Typ == enc.TypeVersionNameComponent {
		// version is sequence number for snapshot
		seqNo = name[len(name)-1]
	} else if seqNo.Typ != enc.TypeSequenceNumNameComponent {
		// version is immutable, sequence number is in name
		log.Warnf("prefixDataFetch: unexpected sequence number type: %s", seqNo.Typ)
		return
	}

	// Update the prefix table
	router.Known = seqNo.NumberVal()
	if dv.pfx.Apply(ops) {
		// Update the local fib if prefix table changed
		go dv.fibUpdate() // very expensive
	}
}
