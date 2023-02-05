/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package fw

import (
	"encoding/binary"
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"github.com/cespare/xxhash"
	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/dispatch"
	"github.com/named-data/YaNFD/ndn"
	"github.com/named-data/YaNFD/table"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

// MaxFwThreads Maximum number of forwarding threads
const MaxFwThreads = 32

// Threads contains all forwarding threads
var Threads map[int]*Thread

// HashNameToFwThread hashes an NDN name to a forwarding thread.
func HashNameToFwThread(name *enc.Name) int {
	// Dispatch all management requests to thread 0
	if len(*name) > 0 && (*name)[0].String() == "localhost" {
		return 0
	}

	return int(xxhash.Sum64String(name.String()) % uint64(len(Threads)))
}

// HashNameToAllPrefixFwThreads hahes an NDN name to all forwarding threads for all prefixes of the name.
func HashNameToAllPrefixFwThreads(name *enc.Name) []int {
	// Dispatch all management requests to thread 0
	if len(*name) > 0 && (*name)[0].String() == "localhost" {
		return []int{0}
	}

	threadMap := make(map[int]interface{})

	// Strings are likely better to work with for performance here than calling Name.prefix
	for nameString := name.String(); len(nameString) > 1; nameString = nameString[:strings.LastIndex(nameString, "/")] {
		threadMap[int(xxhash.Sum64String(nameString)%uint64(len(Threads)))] = true
	}

	threadList := make([]int, 0, len(threadMap))
	for i := range threadMap {
		threadList = append(threadList, i)
	}
	return threadList
}

// Thread Represents a forwarding thread
type Thread struct {
	threadID         int
	pendingInterests chan *ndn.PendingPacket
	pendingDatas     chan *ndn.PendingPacket
	pitCS            table.PitCsTable
	strategies       map[string]Strategy
	deadNonceList    *table.DeadNonceList
	shouldQuit       chan interface{}
	HasQuit          chan interface{}

	// Counters
	NInInterests          uint64
	NInData               uint64
	NOutInterests         uint64
	NOutData              uint64
	NSatisfiedInterests   uint64
	NUnsatisfiedInterests uint64
}

// NewThread creates a new forwarding thread
func NewThread(id int) *Thread {
	t := new(Thread)
	t.threadID = id
	t.pendingInterests = make(chan *ndn.PendingPacket, fwQueueSize)
	t.pendingDatas = make(chan *ndn.PendingPacket, fwQueueSize)
	t.pitCS = table.NewPitCS(t.finalizeInterest)
	t.strategies = InstantiateStrategies(t)
	t.deadNonceList = table.NewDeadNonceList()
	t.shouldQuit = make(chan interface{}, 1)
	t.HasQuit = make(chan interface{})
	return t
}

func (t *Thread) String() string {
	return "FwThread-" + strconv.Itoa(t.threadID)
}

// GetID returns the ID of the forwarding thread
func (t *Thread) GetID() int {
	return t.threadID
}

// GetNumPitEntries returns the number of entries in this thread's PIT.
func (t *Thread) GetNumPitEntries() int {
	return t.pitCS.PitSize()
}

// GetNumCsEntries returns the number of entries in this thread's ContentStore.
func (t *Thread) GetNumCsEntries() int {
	return t.pitCS.CsSize()
}

// TellToQuit tells the forwarding thread to quit
func (t *Thread) TellToQuit() {
	core.LogInfo(t, "Told to quit")
	t.shouldQuit <- true
}

// Run forwarding thread
func (t *Thread) Run() {
	if lockThreadsToCores {
		runtime.LockOSThread()
	}

	pitUpdateTimer := t.pitCS.UpdateTimer()
	for !core.ShouldQuit {
		select {
		case pendingPacket := <-t.pendingInterests:
			t.processIncomingInterest(pendingPacket)
		case pendingPacket := <-t.pendingDatas:
			t.processIncomingData(pendingPacket)
		case <-t.deadNonceList.Ticker.C:
			t.deadNonceList.RemoveExpiredEntries()
		case <-pitUpdateTimer:
			t.pitCS.Update()
		case <-t.shouldQuit:
			continue
		}
	}

	t.deadNonceList.Ticker.Stop()

	core.LogInfo(t, "Stopping thread")
	t.HasQuit <- true
}

// QueueInterest queues an Interest for processing by this forwarding thread.
func (t *Thread) QueueInterest(interest *ndn.PendingPacket) {
	select {
	case t.pendingInterests <- interest:
	default:
		core.LogError(t, "Interest dropped due to full queue")
	}
}

// QueueData queues a Data packet for processing by this forwarding thread.
func (t *Thread) QueueData(data *ndn.PendingPacket) {
	select {
	case t.pendingDatas <- data:
	default:
		core.LogError(t, "Data dropped due to full queue")
	}
}

func (t *Thread) processIncomingInterest(pendingPacket *ndn.PendingPacket) {
	// Ensure incoming face is indicated
	if pendingPacket.IncomingFaceID == nil {
		core.LogError(t, "Interest missing IncomingFaceId - DROP")
		return
	}
	fmt.Println(pendingPacket.NameCache)

	// Already asserted that this is an Interest in link service
	// this is where we convert it into yanfd interest
	// now, we have go-ndn interest spec 2022 struct in pending packet, which is better/less copying
	var interest *ndn.Interest
	//interest = pendingPacket.NetPacket.(*ndn.Interest)
	interest = nil
	// fmt.Printf("%+v\n", pendingPacket.TestPktStruct.Interest)
	// fmt.Printf("%+v\n", interest)
	// fmt.Println("----")
	//fmt.Println(interest.Name())
	//some testing shows this is the exact same as the interest name in go-ndn
	// Get incoming face
	incomingFace := dispatch.GetFace(*pendingPacket.IncomingFaceID)
	if incomingFace == nil {
		core.LogError(t, "Non-existent incoming FaceID=", *pendingPacket.IncomingFaceID, " for Interest=", pendingPacket.NameCache, " - DROP")
		return
	}
	// Get PIT token (if any)
	incomingPitToken := make([]byte, 0)
	if len(pendingPacket.PitToken) > 0 {
		incomingPitToken = make([]byte, len(pendingPacket.PitToken))
		copy(incomingPitToken, pendingPacket.PitToken)
		core.LogTrace(t, "OnIncomingInterest: ", pendingPacket.NameCache, ", FaceID=", incomingFace.FaceID(), ", Has PitToken")
	} else {
		core.LogTrace(t, "OnIncomingInterest: ", pendingPacket.NameCache, ", FaceID=", incomingFace.FaceID())
	}

	// Check if violates /localhost
	if incomingFace.Scope() == ndn.NonLocal && len(pendingPacket.NameCache) > 0 && pendingPacket.TestPktStruct.Interest.NameV[0].String() == "localhost" {
		core.LogWarn(t, "Interest ", pendingPacket.NameCache, " from non-local face=", incomingFace.FaceID(), " violates /localhost scope - DROP")
		return
	}

	t.NInInterests++

	// Check for forwarding hint and, if present, determine if reaching producer region (and then strip forwarding hint)
	var fhName *ndn.Name
	fhName = nil

	// Check if any matching PIT entries (and if duplicate)
	//read into this, looks like this one will have to be manually changed
	pitEntry, isDuplicate := t.pitCS.InsertInterest(pendingPacket, nil, fhName, incomingFace.FaceID())
	if isDuplicate {
		// Interest loop - since we don't use Nacks, just drop
		core.LogInfo(t, "Interest ", pendingPacket.NameCache, " is looping - DROP")
		return
	}
	core.LogDebug(t, "Found or updated PIT entry for Interest=", pendingPacket.NameCache, ", PitToken=", uint64(pitEntry.Token()))

	// Get strategy for name
	// getting strategy for name seems generic enough that it will be easy
	//strategyName := table.FibStrategyTable.FindStrategy(interest.Name())
	strategyName, _ := ndn.NameFromString("/localhost/nfd/strategy/best-route/v=1")
	strategy := t.strategies[strategyName.String()]
	core.LogDebug(t, "Using Strategy=", strategyName, " for Interest=", pendingPacket.NameCache)

	// Add in-record and determine if already pending
	// this looks like custom interest again, but again can be changed without much issue?
	_, isAlreadyPending := pitEntry.InsertInRecord(pendingPacket, interest, incomingFace.FaceID(), incomingPitToken)

	if !isAlreadyPending {
		core.LogTrace(t, "Interest ", pendingPacket.NameCache, " is not pending")

		// Check CS for matching entry
		//need to change this as well
		//if t.pitCS.IsCsServing() {
		if !true {
			csEntry := t.pitCS.FindMatchingDataFromCS(pendingPacket, interest)
			if csEntry != nil {
				// Pass to strategy AfterContentStoreHit pipeline
				strategy.AfterContentStoreHit(pendingPacket, pitEntry, incomingFace.FaceID(), csEntry.Data())
				return
			}
		}
	} else {
		core.LogTrace(t, "Interest ", pendingPacket.NameCache, " is already pending")
	}

	// Update PIT entry expiration timer
	table.UpdateExpirationTimer(pitEntry)
	// pitEntry.UpdateExpirationTimer()

	// If NextHopFaceId set, forward to that face (if it exists) or drop
	if pendingPacket.NextHopFaceID != nil {
		if dispatch.GetFace(*pendingPacket.NextHopFaceID) != nil {
			core.LogTrace(t, "NextHopFaceId is set for Interest ", pendingPacket.NameCache, " - dispatching directly to face")
			dispatch.GetFace(*pendingPacket.NextHopFaceID).SendPacket(pendingPacket)
		} else {
			core.LogInfo(t, "Non-existent face specified in NextHopFaceId for Interest ", pendingPacket.NameCache, " - DROP")
		}
		return
	}

	// Pass to strategy AfterReceiveInterest pipeline
	var trash []*table.FibNextHopEntry
	var nexthop []*table.FibNextHopEntry
	if fhName == nil {
		// for _, name := range pendingPacket.TestPktStruct.Interest.NameV {
		// 	fmt.Println(name)
		// }
		// fmt.Println(interest.Name())
		// for i := 0; i < interest.Name().Size(); i++ {
		// 	fmt.Println(interest.Name().At(i))
		// }

		trash = table.FibStrategyTable.FindNextHops1(&pendingPacket.TestPktStruct.Interest.NameV)
		//nexthop = table.FibStrategyTable.FindNextHops(interest.Name())
		if len(nexthop) > 0 {
			fmt.Println(nexthop[0])
		}
		if len(trash) > 0 {
			fmt.Println(trash[0])
		}
	} else {
		//trash = table.FibStrategyTable.FindNextHops(fhName)
	}
	strategy.AfterReceiveInterest(pendingPacket, pitEntry, incomingFace.FaceID(), interest, trash)
	//strategy.AfterReceiveInterest(pendingPacket, pitEntry, incomingFace.FaceID(), interest, nexthop)
}

func (t *Thread) processOutgoingInterest(pp *ndn.PendingPacket, interest *ndn.Interest, pitEntry table.PitEntry, nexthop uint64, inFace uint64) bool {
	core.LogTrace(t, "OnOutgoingInterest: ", pp.NameCache, ", FaceID=", nexthop)

	// Get outgoing face
	outgoingFace := dispatch.GetFace(nexthop)
	if outgoingFace == nil {
		core.LogError(t, "Non-existent nexthop FaceID=", nexthop, " for Interest=", interest.Name(), " - DROP")
		return false
	}
	if outgoingFace.FaceID() == inFace && outgoingFace.LinkType() != ndn.AdHoc {
		core.LogDebug(t, "Attempting to send Interest=", pp.NameCache, " back to incoming face - DROP")
		return false
	}

	// Drop if HopLimit (if present) on Interest going to non-local face is 0. If so, drop
	if pp.TestPktStruct.Interest.HopLimitV != nil && int(*pp.TestPktStruct.Interest.HopLimitV) == 0 && outgoingFace.Scope() == ndn.NonLocal {
		core.LogDebug(t, "Attempting to send Interest=", pp.NameCache, " with HopLimit=0 to non-local face - DROP")
		return false
	}

	// Create or update out-record
	pitEntry.InsertOutRecord(pp, interest, nexthop)

	t.NOutInterests++

	// Send on outgoing face
	pp.IncomingFaceID = new(uint64)
	*pp.IncomingFaceID = uint64(inFace)
	pp.PitToken = make([]byte, 6)
	binary.BigEndian.PutUint16(pp.PitToken, uint16(t.threadID))
	binary.BigEndian.PutUint32(pp.PitToken[2:], pitEntry.Token())
	outgoingFace.SendPacket(pp)
	return true
}

func (t *Thread) finalizeInterest(pitEntry table.PitEntry) {
	core.LogTrace(t, "OnFinalizeInterest: ", pitEntry.Name())

	// Check for nonces to insert into dead nonce list
	for _, outRecord := range pitEntry.OutRecords() {
		t.deadNonceList.Insert1(outRecord.LatestPacket.NameCache, outRecord.PacketNonce)
	}

	// Counters
	if !pitEntry.Satisfied() {
		t.NUnsatisfiedInterests += uint64(len(pitEntry.InRecords()))
	}
}

func (t *Thread) processIncomingData(pendingPacket *ndn.PendingPacket) {
	// Ensure incoming face is indicated
	if pendingPacket.IncomingFaceID == nil {
		core.LogError(t, "Data missing IncomingFaceId - DROP")
		return
	}

	// Get PIT if present
	var pitToken *uint32
	if len(pendingPacket.PitToken) > 0 {
		pitToken = new(uint32)
		// We have already guaranteed that, if a PIT token is present, it is 6 bytes long
		*pitToken = binary.BigEndian.Uint32(pendingPacket.PitToken[2:6])
	}

	var data *ndn.Data

	// Get incoming face
	incomingFace := dispatch.GetFace(*pendingPacket.IncomingFaceID)
	if incomingFace == nil {
		core.LogError(t, "Non-existent nexthop FaceID=", *pendingPacket.IncomingFaceID, " for Data=", pendingPacket.NameCache, " DROP")
		return
	}

	t.NInData++

	// Check if violates /localhost
	if incomingFace.Scope() == ndn.NonLocal && len(pendingPacket.NameCache) > 0 && pendingPacket.TestPktStruct.Data.NameV[0].String() == "localhost" {
		core.LogWarn(t, "Data ", pendingPacket.NameCache, " from non-local FaceID=", *pendingPacket.IncomingFaceID, " violates /localhost scope - DROP")
		return
	}

	// Add to Content Store
	if t.pitCS.IsCsAdmitting() {
		t.pitCS.InsertData(pendingPacket, data)
	}

	// Check for matching PIT entries
	pitEntries := t.pitCS.FindInterestPrefixMatchByData1(pendingPacket, data, pitToken)
	if len(pitEntries) == 0 {
		// Unsolicated Data - nothing more to do
		core.LogDebug(t, "Unsolicited data ", pendingPacket.NameCache, " - DROP")
		return
	}
	// Get strategy for name

	//strategyName := table.FibStrategyTable.FindStrategy(data.Name())
	strategyName, _ := ndn.NameFromString("/localhost/nfd/strategy/best-route/v=1")
	strategy := t.strategies[strategyName.String()]

	if len(pitEntries) == 1 {
		// Set PIT entry expiration to now
		table.SetExpirationTimerToNow(pitEntries[0])
		// pitEntries[0].SetExpirationTimerToNow()

		// Invoke strategy's AfterReceiveData
		core.LogTrace(t, "Sending Data=", pendingPacket.NameCache, " to strategy=", strategyName)
		strategy.AfterReceiveData(pendingPacket, pitEntries[0], *pendingPacket.IncomingFaceID, data)

		// Mark PIT entry as satisfied
		pitEntries[0].SetSatisfied(true)

		// Insert into dead nonce list
		for _, outRecord := range pitEntries[0].OutRecords() {
			t.deadNonceList.Insert1(pendingPacket.NameCache, outRecord.PacketNonce)
		}

		// Clear out records from PIT entry
		pitEntries[0].ClearOutRecords()
	} else {
		for _, pitEntry := range pitEntries {
			// Store all pending downstreams (except face Data packet arrived on) and PIT tokens
			downstreams := make(map[uint64][]byte)
			for downstreamFaceID, downstreamFaceRecord := range pitEntry.InRecords() {
				if downstreamFaceID != *pendingPacket.IncomingFaceID {
					// TODO: Ad-hoc faces
					downstreams[downstreamFaceID] = make([]byte, len(downstreamFaceRecord.PitToken))
					copy(downstreams[downstreamFaceID], downstreamFaceRecord.PitToken)
				}
			}

			// Set PIT entry expiration to now
			table.SetExpirationTimerToNow(pitEntry)
			// pitEntry.SetExpirationTimerToNow()f

			// Invoke strategy's BeforeSatisfyInterest
			strategy.BeforeSatisfyInterest(pitEntry, *pendingPacket.IncomingFaceID, data)

			// Mark PIT entry as satisfied
			pitEntry.SetSatisfied(true)

			// Insert into dead nonce list
			for _, outRecord := range pitEntries[0].GetOutRecords() {
				t.deadNonceList.Insert1(pendingPacket.NameCache, outRecord.PacketNonce)
			}

			// Clear PIT entry's in- and out-records
			pitEntry.ClearInRecords()
			pitEntry.ClearOutRecords()

			// Call outoing Data pipeline for each pending downstream
			for downstreamFaceID, downstreamPITToken := range downstreams {
				core.LogTrace(t, "Multiple matching PIT entries for ", pendingPacket.NameCache, ": sending to OnOutgoingData pipeline")
				t.processOutgoingData(pendingPacket, data, downstreamFaceID, downstreamPITToken, *pendingPacket.IncomingFaceID)
			}
		}
	}
}

func (t *Thread) processOutgoingData(pp *ndn.PendingPacket, data *ndn.Data, nexthop uint64, pitToken []byte, inFace uint64) {
	core.LogTrace(t, "OnOutgoingData: ", pp.NameCache, ", FaceID=", nexthop)

	// Get outgoing face
	outgoingFace := dispatch.GetFace(nexthop)
	if outgoingFace == nil {
		core.LogError(t, "Non-existent nexthop FaceID=", nexthop, " for Data=", pp, " - DROP")
		return
	}

	// Check if violates /localhost
	if outgoingFace.Scope() == ndn.NonLocal && len(pp.NameCache) > 0 && pp.TestPktStruct.Data.NameV[0].String() == "localhost" {
		core.LogWarn(t, "Data ", pp.NameCache, " cannot be sent to non-local FaceID=", nexthop, " since violates /localhost scope - DROP")
		return
	}

	t.NOutData++
	t.NSatisfiedInterests++

	// Send on outgoing face
	if len(pitToken) > 0 {
		pp.PitToken = make([]byte, len(pitToken))
		copy(pp.PitToken, pitToken)
	}
	pp.IncomingFaceID = new(uint64)
	*pp.IncomingFaceID = uint64(inFace)
	//fmt.Println("sent packet")
	outgoingFace.SendPacket(pp)
}
