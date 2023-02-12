/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package fw

import (
	"strconv"

	"github.com/named-data/YaNFD/ndn"
	"github.com/named-data/YaNFD/table"
)

// StrategyPrefix is the prefix of all strategy names for YaNFD
const StrategyPrefix = "/localhost/nfd/strategy"

// Strategy represents a forwarding strategy.
type Strategy interface {
	Instantiate(fwThread *Thread)
	String() string
	GetName() *ndn.Name

	AfterContentStoreHit(pp *ndn.PendingPacket, pitEntry table.PitEntry, inFace uint64)
	AfterReceiveData(pp *ndn.PendingPacket, pitEntry table.PitEntry, inFace uint64)
	AfterReceiveInterest(pp *ndn.PendingPacket, pitEntry table.PitEntry, inFace uint64, nexthops []*table.FibNextHopEntry)
	BeforeSatisfyInterest(pitEntry table.PitEntry, inFace uint64)
}

// StrategyBase provides common helper methods for YaNFD forwarding strategies.
type StrategyBase struct {
	thread          *Thread
	threadID        int
	name            *ndn.Name
	strategyName    *ndn.GenericNameComponent
	version         uint64
	strategyLogName string
}

// NewStrategyBase is a helper that allows specific strategies to initialize the base.
func (s *StrategyBase) NewStrategyBase(fwThread *Thread, strategyName *ndn.GenericNameComponent, version uint64, strategyLogName string) {
	s.thread = fwThread
	s.threadID = s.thread.threadID
	s.name, _ = ndn.NameFromString(StrategyPrefix)
	s.strategyName = strategyName
	s.name.Append(strategyName).Append(ndn.NewVersionNameComponent(version))
	s.version = version
	s.strategyLogName = strategyLogName
}

func (s *StrategyBase) String() string {
	return s.strategyLogName + "-v" + strconv.FormatUint(s.version, 10) + "-Thread" + strconv.Itoa(s.threadID)
}

// GetName returns the name of strategy, including version information.
func (s *StrategyBase) GetName() *ndn.Name {
	return s.name
}

// SendInterest sends an Interest on the specified face.
func (s *StrategyBase) SendInterest(pp *ndn.PendingPacket, pitEntry table.PitEntry, nexthop uint64, inFace uint64) bool {
	return s.thread.processOutgoingInterest(pp, pitEntry, nexthop, inFace)
}

// SendData sends a Data packet on the specified face.
func (s *StrategyBase) SendData(pp *ndn.PendingPacket, pitEntry table.PitEntry, nexthop uint64, inFace uint64) {
	var pitToken []byte
	if inRecord, ok := pitEntry.InRecords()[nexthop]; ok {
		pitToken = inRecord.PitToken
		delete(pitEntry.InRecords(), nexthop)
	}
	s.thread.processOutgoingData(pp, nexthop, pitToken, inFace)
}
