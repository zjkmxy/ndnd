/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package fw

import (
	"reflect"
	"sort"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/ndn"
	"github.com/named-data/YaNFD/table"
)

// BestRoute is a forwarding strategy that forwards Interests to the nexthop with the lowest cost.
type BestRoute struct {
	StrategyBase
}

func init() {
	strategyTypes = append(strategyTypes, reflect.TypeOf(new(BestRoute)))
	StrategyVersions["best-route"] = []uint64{1}
}

// Instantiate creates a new instance of the BestRoute strategy.
func (s *BestRoute) Instantiate(fwThread *Thread) {
	s.NewStrategyBase(fwThread, ndn.NewGenericNameComponent([]byte("best-route")), 1, "BestRoute")
}

// AfterContentStoreHit ...
func (s *BestRoute) AfterContentStoreHit(pp *ndn.PendingPacket, pitEntry table.PitEntry, inFace uint64, data *ndn.Data) {
	// Send downstream
	core.LogTrace(s, "AfterContentStoreHit: Forwarding content store hit Data=", pp.NameCache, " to FaceID=", inFace)
	s.SendData(pp, data, pitEntry, inFace, 0) // 0 indicates ContentStore is source
}

// AfterReceiveData ...
func (s *BestRoute) AfterReceiveData(pp *ndn.PendingPacket, pitEntry table.PitEntry, inFace uint64, data *ndn.Data) {
	core.LogTrace(s, "AfterReceiveData: Data=", pp.NameCache, ", ", len(pitEntry.InRecords()), " In-Records")
	for faceID := range pitEntry.InRecords() {
		core.LogTrace(s, "AfterReceiveData: Forwarding Data=", pp.NameCache, " to FaceID=", faceID)
		s.SendData(pp, data, pitEntry, faceID, inFace)
	}
}

// AfterReceiveInterest ...
func (s *BestRoute) AfterReceiveInterest(pp *ndn.PendingPacket, pitEntry table.PitEntry, inFace uint64, interest *ndn.Interest, nexthops []*table.FibNextHopEntry) {
	sort.Slice(nexthops, func(i, j int) bool { return nexthops[i].Cost < nexthops[j].Cost })
	for _, nh := range nexthops {
		core.LogTrace(s, "AfterReceiveInterest: Forwarding Interest=", pp.NameCache, " to FaceID=", nh.Nexthop)
		if sent := s.SendInterest(pp, interest, pitEntry, nh.Nexthop, inFace); sent {
			return
		}
	}

	core.LogDebug(s, "AfterReceiveInterest: No usable nexthop for Interest=", pp.NameCache, " - DROP")
}

// BeforeSatisfyInterest ...
func (s *BestRoute) BeforeSatisfyInterest(pitEntry table.PitEntry, inFace uint64, data *ndn.Data) {
	// This does nothing in BestRoute
}
