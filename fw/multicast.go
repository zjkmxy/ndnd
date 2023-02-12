/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package fw

import (
	"reflect"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/ndn"
	"github.com/named-data/YaNFD/table"
)

// Multicast is a forwarding strategy that forwards Interests to all nexthop faces.
type Multicast struct {
	StrategyBase
}

func init() {
	strategyTypes = append(strategyTypes, reflect.TypeOf(new(Multicast)))
	StrategyVersions["multicast"] = []uint64{1}
}

// Instantiate creates a new instance of the Multicast strategy.
func (s *Multicast) Instantiate(fwThread *Thread) {
	s.NewStrategyBase(fwThread, ndn.NewGenericNameComponent([]byte("multicast")), 1, "Multicast")
}

// AfterContentStoreHit ...
func (s *Multicast) AfterContentStoreHit(pp *ndn.PendingPacket, pitEntry table.PitEntry, inFace uint64) {
	// Send downstream
	core.LogTrace(s, "AfterContentStoreHit: Forwarding content store hit Data=", " to FaceID=", inFace)
	s.SendData(pp, pitEntry, inFace, 0) // 0 indicates ContentStore is source
}

// AfterReceiveData ...
func (s *Multicast) AfterReceiveData(pp *ndn.PendingPacket, pitEntry table.PitEntry, inFace uint64) {
	core.LogTrace(s, "AfterReceiveData: Data=", pp.TestPktStruct.Data.NameV, ", ", len(pitEntry.InRecords()), " In-Records")
	for faceID := range pitEntry.InRecords() {
		core.LogTrace(s, "AfterReceiveData: Forwarding Data=", pp.TestPktStruct.Data.NameV, " to FaceID=", faceID)
		s.SendData(pp, pitEntry, faceID, inFace)
	}
}

// AfterReceiveInterest ...
func (s *Multicast) AfterReceiveInterest(pp *ndn.PendingPacket, pitEntry table.PitEntry, inFace uint64, nexthops []*table.FibNextHopEntry) {
	if len(nexthops) == 0 {
		core.LogDebug(s, "AfterReceiveInterest: No nexthop for Interest=", " - DROP")
		return
	}

	for _, nexthop := range nexthops {
		core.LogTrace(s, "AfterReceiveInterest: Forwarding Interest=", " to FaceID=", nexthop.Nexthop)
		s.SendInterest(pp, pitEntry, nexthop.Nexthop, inFace)
	}
}

// BeforeSatisfyInterest ...
func (s *Multicast) BeforeSatisfyInterest(pitEntry table.PitEntry, inFace uint64) {
	// This does nothing in Multicast
}
