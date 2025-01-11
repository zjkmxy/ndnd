package dv

import (
	"time"

	"github.com/named-data/ndnd/dv/table"
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	spec "github.com/named-data/ndnd/std/ndn/spec_2022"
	spec_svs "github.com/named-data/ndnd/std/ndn/svs/v3"
	sec "github.com/named-data/ndnd/std/security"
	"github.com/named-data/ndnd/std/utils"
)

func (dv *Router) advertSyncSendInterest() (err error) {
	// Sync Interests for our outgoing connections
	err = dv.advertSyncSendInterestImpl(dv.config.AdvertisementSyncActivePrefix())
	if err != nil {
		log.Warnf("advert-sync: failed to send active sync interest: %+v", err)
	}

	// Sync Interests for incoming connections
	err = dv.advertSyncSendInterestImpl(dv.config.AdvertisementSyncPassivePrefix())
	if err != nil {
		log.Warnf("advert-sync: failed to send passive sync interest: %+v", err)
	}

	return err
}

func (dv *Router) advertSyncSendInterestImpl(prefix enc.Name) (err error) {
	// SVS v3 Sync Data
	syncName := prefix.Append(enc.NewVersionComponent(3))

	// State Vector for our group
	sv := &spec_svs.SvsData{
		StateVector: &spec_svs.StateVector{
			Entries: []*spec_svs.StateVectorEntry{{
				Name: dv.config.RouterName(),
				SeqNoEntries: []*spec_svs.SeqNoEntry{{
					BootstrapTime: dv.advertBootTime,
					SeqNo:         dv.advertSyncSeq,
				}},
			}},
		},
	}

	// TODO: sign the sync data
	signer := sec.NewSha256Signer()

	dataCfg := &ndn.DataConfig{
		ContentType: utils.IdPtr(ndn.ContentTypeBlob),
	}
	data, err := dv.engine.Spec().MakeData(syncName, dataCfg, sv.Encode(), signer)
	if err != nil {
		log.Errorf("advert-sync: sendSyncInterest failed make data: %+v", err)
		return
	}

	// Make SVS Sync Interest
	intCfg := &ndn.InterestConfig{
		Lifetime: utils.IdPtr(1 * time.Second),
		Nonce:    utils.ConvertNonce(dv.engine.Timer().Nonce()),
		HopLimit: utils.IdPtr(uint(2)), // use localhop w/ this
	}
	interest, err := dv.engine.Spec().MakeInterest(syncName, intCfg, data.Wire, nil)
	if err != nil {
		return err
	}

	// Sync Interest has no reply
	err = dv.engine.Express(interest, nil)
	if err != nil {
		return err
	}

	return nil
}

func (dv *Router) advertSyncOnInterest(args ndn.InterestHandlerArgs, active bool) {
	// If there is no incoming face ID, we can't use this
	if args.IncomingFaceId == nil {
		log.Warn("advert-sync: received Sync Interest with no incoming face ID, ignoring")
		return
	}

	// Check if app param is present
	if args.Interest.AppParam() == nil {
		log.Warn("advert-sync: received Sync Interest with no AppParam, ignoring")
		return
	}

	// Decode Sync Data
	pkt, _, err := spec.ReadPacket(enc.NewWireReader(args.Interest.AppParam()))
	if err != nil {
		log.Warnf("advert-sync: failed to parse Sync Data: %+v", err)
		return
	}
	if pkt.Data == nil {
		log.Warnf("advert-sync: no Sync Data, ignoring")
		return
	}

	// TODO: verify signature on Sync Interest

	// Decode state vector
	svWire := pkt.Data.Content()
	params, err := spec_svs.ParseSvsData(enc.NewWireReader(svWire), false)
	if err != nil || params.StateVector == nil {
		log.Warnf("advert-sync: failed to parse StateVec: %+v", err)
		return
	}

	// Process each entry in the state vector
	dv.mutex.Lock()
	defer dv.mutex.Unlock()

	// FIB needs update if face changes for any neighbor
	fibDirty := false
	markRecvPing := func(ns *table.NeighborState) {
		err, faceDirty := ns.RecvPing(*args.IncomingFaceId, active)
		if err != nil {
			log.Warnf("advert-sync: failed to update neighbor: %+v", err)
		}
		fibDirty = fibDirty || faceDirty
	}

	// There should only be one entry in the StateVector, but check all anyway
	for _, node := range params.StateVector.Entries {
		if len(node.SeqNoEntries) != 1 {
			log.Warnf("advert-sync: unexpected %d SeqNoEntries for %s, ignoring", len(node.SeqNoEntries), node.Name)
			return
		}
		entry := node.SeqNoEntries[0]

		// Parse name from entry
		if node.Name == nil {
			log.Warnf("advert-sync: failed to parse neighbor name: %+v", err)
			continue
		}

		// Check if the entry is newer than what we know
		ns := dv.neighbors.Get(node.Name)
		if ns != nil {
			if ns.AdvertBoot >= entry.BootstrapTime && ns.AdvertSeq >= entry.SeqNo {
				// Nothing has changed, skip
				markRecvPing(ns)
				continue
			}
		} else {
			// Create new neighbor entry cause none found
			// This is the ONLY place where neighbors are created
			// In all other places, quit if not found
			ns = dv.neighbors.Add(node.Name)
		}

		markRecvPing(ns)
		ns.AdvertBoot = entry.BootstrapTime
		ns.AdvertSeq = entry.SeqNo

		go dv.advertDataFetch(node.Name, entry.BootstrapTime, entry.SeqNo)
	}

	// Update FIB if needed
	if fibDirty {
		go dv.fibUpdate()
	}
}
