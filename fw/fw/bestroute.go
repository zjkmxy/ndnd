/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package fw

import (
	"sort"
	"time"

	"github.com/named-data/ndnd/fw/core"
	"github.com/named-data/ndnd/fw/defn"
	"github.com/named-data/ndnd/fw/table"
)

// BestRouteSuppressionTime is the time to suppress retransmissions of the same Interest.
const BestRouteSuppressionTime = 500 * time.Millisecond

// BestRoute is a forwarding strategy that forwards Interests
// to the nexthop with the lowest cost.
type BestRoute struct {
	StrategyBase
}

func init() {
	strategyInit = append(strategyInit, func() Strategy { return &BestRoute{} })
	StrategyVersions["best-route"] = []uint64{1}
}

func (s *BestRoute) Instantiate(fwThread *Thread) {
	s.NewStrategyBase(fwThread, "best-route", 1)
}

func (s *BestRoute) AfterContentStoreHit(
	packet *defn.Pkt,
	pitEntry table.PitEntry,
	inFace uint64,
) {
	core.Log.Trace(s, "AfterContentStoreHit", "name", packet.Name, "faceid", inFace)
	s.SendData(packet, pitEntry, inFace, 0) // 0 indicates ContentStore is source
}

func (s *BestRoute) AfterReceiveData(
	packet *defn.Pkt,
	pitEntry table.PitEntry,
	inFace uint64,
) {
	core.Log.Trace(s, "AfterReceiveData", "name", packet.Name, "inrecords", len(pitEntry.InRecords()))
	for faceID := range pitEntry.InRecords() {
		core.Log.Trace(s, "Forwarding Data", "name", packet.Name, "faceid", faceID)
		s.SendData(packet, pitEntry, faceID, inFace)
	}
}

func (s *BestRoute) AfterReceiveInterest(
	packet *defn.Pkt,
	pitEntry table.PitEntry,
	inFace uint64,
	nexthops []*table.FibNextHopEntry,
) {
	if len(nexthops) == 0 {
		core.Log.Debug(s, "No nexthop found - DROP", "name", packet.Name)
		return
	}

	// If there is an out record less than suppression interval ago, drop the
	// retransmission to suppress it (only if the nonce is different)
	for _, outRecord := range pitEntry.OutRecords() {
		if outRecord.LatestNonce != packet.L3.Interest.NonceV.Unwrap() &&
			outRecord.LatestTimestamp.Add(BestRouteSuppressionTime).After(time.Now()) {
			core.Log.Debug(s, "Suppressed Interest - DROP", "name", packet.Name)
			return
		}
	}

	// Sort nexthops by cost and send to best-possible nexthop
	sort.Slice(nexthops, func(i, j int) bool { return nexthops[i].Cost < nexthops[j].Cost })
	for _, nh := range nexthops {
		core.Log.Trace(s, "Forwarding Interest", "name", packet.Name, "faceid", nh.Nexthop)
		if sent := s.SendInterest(packet, pitEntry, nh.Nexthop, inFace); sent {
			return
		}
	}

	core.Log.Debug(s, "No usable nexthop for Interest - DROP", "name", packet.Name)
}

func (s *BestRoute) BeforeSatisfyInterest(pitEntry table.PitEntry, inFace uint64) {
	// This does nothing in BestRoute
}
