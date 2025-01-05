/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2022 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"math/rand"
	"time"

	"github.com/named-data/ndnd/fw/core"
	"github.com/named-data/ndnd/fw/face"
	"github.com/named-data/ndnd/fw/table"
	enc "github.com/named-data/ndnd/std/encoding"
	basic_engine "github.com/named-data/ndnd/std/engine/basic"
	"github.com/named-data/ndnd/std/ndn"
	mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
	spec "github.com/named-data/ndnd/std/ndn/spec_2022"
	sec "github.com/named-data/ndnd/std/security"
	"github.com/named-data/ndnd/std/utils"
)

var NFD_COMP = enc.NewStringComponent(enc.TypeGenericNameComponent, "nfd")
var LOCAL_PREFIX = enc.Name{enc.LOCALHOST, NFD_COMP}
var NON_LOCAL_PREFIX = enc.Name{enc.LOCALHOP, NFD_COMP}

// Thread Represents the management thread
type Thread struct {
	face       face.LinkService
	transport  *face.InternalTransport
	modules    map[string]Module
	timer      ndn.Timer
	datasetVer uint64
}

func (m *Thread) String() string {
	return "Mgmt"
}

// MakeMgmtThread creates a new management thread.
func MakeMgmtThread() *Thread {
	m := new(Thread)
	m.timer = basic_engine.NewTimer()

	m.modules = make(map[string]Module)
	m.registerModule("cs", new(ContentStoreModule))
	m.registerModule("faces", new(FaceModule))
	m.registerModule("fib", new(FIBModule))
	m.registerModule("rib", new(RIBModule))
	m.registerModule("status", new(ForwarderStatusModule))
	m.registerModule("strategy-choice", new(StrategyChoiceModule))

	// readvertisers run in the management thread for ease of
	// implementation, since they use the internal transport
	if core.GetConfig().Tables.Rib.ReadvertiseNlsr {
		table.AddReadvertiser(NewNlsrReadvertiser(m))
	}

	return m
}

func (m *Thread) registerModule(name string, module Module) {
	m.modules[name] = module
	module.registerManager(m)
}

// Run management thread
func (m *Thread) Run() {
	core.LogInfo(m, "Starting management thread")

	// Create and register Internal transport
	m.face, m.transport = face.RegisterInternalTransport()
	table.FibStrategyTable.InsertNextHopEnc(LOCAL_PREFIX, m.face.FaceID(), 0)
	if enableLocalhopManagement {
		table.FibStrategyTable.InsertNextHopEnc(NON_LOCAL_PREFIX, m.face.FaceID(), 0)
	}

	for {
		lpPkt := m.transport.Receive()
		if lpPkt == nil {
			// Indicates that internal face has quit, which means it's time for us to quit
			core.LogInfo(m, "Face quit, so management quitting")
			break
		}

		if lpPkt.IncomingFaceId == nil || len(lpPkt.Fragment) == 0 {
			core.LogWarn(m, "Received malformed packet on internal face, drop")
			continue
		}

		pkt, _, err := spec.ReadPacket(enc.NewWireReader(lpPkt.Fragment))
		if err != nil {
			core.LogWarn(m, "Unable to decode internal packet, drop")
			continue
		}

		// We only expect Interests, so drop Data packets
		if pkt.Interest == nil {
			core.LogDebug(m, "Dropping received non-Interest packet")
			continue
		}

		// Create internal Interest object for easier handling
		interest := &Interest{
			Interest: *pkt.Interest,
			pitToken: lpPkt.PitToken,
			inFace:   lpPkt.IncomingFaceId,
		}

		// Ensure Interest name matches expectations
		if len(interest.Name()) < len(LOCAL_PREFIX)+2 { // Module + Verb
			core.LogWarn(m, "Control command name ", interest.Name(), " has unexpected number of components - DROP")
			continue
		}
		if !LOCAL_PREFIX.IsPrefix(interest.Name()) && !NON_LOCAL_PREFIX.IsPrefix(interest.Name()) {
			core.LogWarn(m, "Control command name ", interest.Name(), " has unexpected prefix - DROP")
			continue
		}

		core.LogTrace(m, "Received management Interest ", interest.Name())

		// Dispatch interest based on name
		moduleName := interest.Name()[len(LOCAL_PREFIX)].String()
		if module, ok := m.modules[moduleName]; ok {
			module.handleIncomingInterest(interest)
		} else {
			core.LogWarn(m, "Received management Interest for unknown module ", moduleName)
			m.sendCtrlResp(interest, 501, "Unknown module", nil)
		}
	}
}

func (m *Thread) sendInterest(name enc.Name, params enc.Wire) {
	config := ndn.InterestConfig{
		MustBeFresh: true,
		Nonce:       utils.IdPtr(rand.Uint64()),
	}
	interest, err := spec.Spec{}.MakeInterest(name, &config, params, sec.NewSha256IntSigner(m.timer))
	if err != nil {
		core.LogWarn(m, "Unable to encode Interest for ", name, ": ", err)
		return
	}

	m.transport.Send(&spec.LpPacket{Fragment: interest.Wire})
	core.LogTrace(m, "Sent management Interest for ", interest.FinalName)
}

func (m *Thread) sendData(interest *Interest, name enc.Name, content enc.Wire) {
	data, err := spec.Spec{}.MakeData(name,
		&ndn.DataConfig{
			ContentType: utils.IdPtr(ndn.ContentTypeBlob),
			Freshness:   utils.IdPtr(time.Second),
		},
		content,
		sec.NewSha256Signer(),
	)
	if err != nil {
		core.LogWarn(m, "Unable to encode Data for ", interest.Name(), ": ", err)
		return
	}

	m.transport.Send(&spec.LpPacket{
		Fragment:      data.Wire,
		PitToken:      interest.pitToken,
		NextHopFaceId: interest.inFace,
	})
	core.LogTrace(m, "Sent management Data for ", name)
}

func (m *Thread) sendCtrlResp(interest *Interest, statusCode uint64, statusText string, params *mgmt.ControlArgs) {
	if params == nil {
		params = &mgmt.ControlArgs{}
	}

	res := &mgmt.ControlResponse{
		Val: &mgmt.ControlResponseVal{
			StatusCode: statusCode,
			StatusText: statusText,
			Params:     params,
		},
	}

	m.sendData(interest, interest.Name(), res.Encode())
}

func (m *Thread) sendStatusDataset(interest *Interest, name enc.Name, dataset enc.Wire) {
	// TODO: support segmented datasets
	m.datasetVer++
	segments := makeStatusDataset(name, m.datasetVer, dataset)

	m.transport.Send(&spec.LpPacket{
		Fragment:      segments,
		PitToken:      interest.pitToken,
		NextHopFaceId: interest.inFace,
	})
}
