package ackconn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"os"
	"sort"
	"time"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/dispatch"
	"github.com/named-data/YaNFD/face"
	"github.com/named-data/YaNFD/fw"
	"github.com/named-data/YaNFD/ndn"
	"github.com/named-data/YaNFD/ndn/mgmt"
	"github.com/named-data/YaNFD/ndn/tlv"
	"github.com/named-data/YaNFD/table"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

var AckChannel AckConn

type AckConn struct {
	conn       net.Conn
	socketFile string
}
type Message struct {
	Command         string                 `json:"command"`
	Name            string                 `json:"name"`
	ParamName       string                 `json:"paramname"`
	FaceID          uint64                 `json:"faceid"`
	Cost            uint64                 `json:"cost"`
	Strategy        string                 `json:"strategy"`
	Capacity        int                    `json:"capacity"`
	Versions        []uint64               `json:"versions"`
	Dataset         []byte                 `json:"dataset"`
	Valid           bool                   `json:"valid"`
	ControlParams   mgmt.ControlParameters `json:"controlparams"`
	ControlResponse mgmt.ControlResponse   `json:"controlresponse"`
	ErrorCode       int                    `json:"errorcode"`
	ErrorMessage    string                 `json:"errormessage"`
	ParamsValid     bool                   `json:"paramsvalid"`
	FaceQueryFilter mgmt.FaceQueryFilter   `json:"facequeryfilter"`
}

func MakeAck() Message {
	msg := Message{
		Valid: true,
	}
	return msg
}

func MakeError(errorcode int) Message {
	msg := Message{
		ErrorCode: errorcode,
	}
	return msg
}

func (a *AckConn) Make(socketFile string) {
	a.socketFile = socketFile
}
func (a *AckConn) RunReceive() {
	// listen to incoming unix packets
	os.Remove(a.socketFile)
	listener, err := net.Listen("unixpacket", a.socketFile)
	if err := os.Chmod(a.socketFile, 0777); err != nil {
		fmt.Println(err)
	}
	if err != nil {
		return
	}
	defer listener.Close()
	a.conn, _ = listener.Accept()
	for {
		buf := make([]byte, 8800)
		size, err := a.conn.Read(buf)
		if err != nil {
			continue
		}
		a.process(size, buf)
	}
}
func (a *AckConn) process(size int, buf []byte) {
	//var response string = "test"
	buf = bytes.Trim(buf, "\x00")
	var commands Message
	err := json.Unmarshal(buf, &commands)
	if err != nil {
		fmt.Println("error:", err)
	}
	switch commands.Command {
	case "list":
		entries := table.FibStrategyTable.GetAllFIBEntries()
		dataset := make([]byte, 0)
		for _, fsEntry := range entries {
			fibEntry := mgmt.MakeFibEntry(fsEntry.Name())
			for _, nexthop := range fsEntry.GetNextHops() {
				var record mgmt.NextHopRecord
				record.FaceID = nexthop.Nexthop
				record.Cost = nexthop.Cost
				fibEntry.Nexthops = append(fibEntry.Nexthops, record)
			}

			wire, err := fibEntry.Encode()
			if err != nil {
				continue
			}
			encoded, err := wire.Wire()
			if err != nil {
				continue
			}
			dataset = append(dataset, encoded...)
		}
		msg := Message{
			Dataset: dataset,
		}
		a.sendMessage(msg)
	case "forwarderstatus":
		status := mgmt.MakeGeneralStatus()
		status.NfdVersion = core.Version
		status.StartTimestamp = uint64(core.StartTimestamp.UnixNano() / 1000 / 1000)
		status.CurrentTimestamp = uint64(time.Now().UnixNano() / 1000 / 1000)
		status.NFibEntries = uint64(len(table.FibStrategyTable.GetAllFIBEntries()))
		for threadID := 0; threadID < fw.NumFwThreads; threadID++ {
			thread := dispatch.GetFWThread(threadID)
			status.NPitEntries += uint64(thread.GetNumPitEntries())
			status.NCsEntries += uint64(thread.GetNumCsEntries())
			status.NInInterests += thread.(*fw.Thread).NInInterests
			status.NInData += thread.(*fw.Thread).NInData
			status.NOutInterests += thread.(*fw.Thread).NOutInterests
			status.NOutData += thread.(*fw.Thread).NOutData
			status.NSatisfiedInterests += thread.(*fw.Thread).NSatisfiedInterests
			status.NUnsatisfiedInterests += thread.(*fw.Thread).NUnsatisfiedInterests
		}
		wire, err := status.Encode()
		if err != nil {
			return
		}
		dataset := wire.Value()
		msg := Message{
			Dataset: dataset,
		}
		a.sendMessage(msg)
	case "faceid":
		faceID := commands.FaceID
		var msg Message
		if face.FaceTable.Get(uint64(faceID)) != nil {
			msg = Message{
				Valid: true,
			}

		} else {
			msg = Message{
				Valid: false,
			}
		}
		a.sendMessage(msg)
	case "liststrategy":
		entries := table.FibStrategyTable.GetAllForwardingStrategies()
		dataset := make([]byte, 0)
		strategyChoiceList := mgmt.MakeStrategyChoiceList()
		for _, fsEntry := range entries {
			strategyChoiceList = append(strategyChoiceList, mgmt.MakeStrategyChoice(fsEntry.Name(), fsEntry.GetStrategy()))
		}

		wires, err := strategyChoiceList.Encode()
		if err != nil {
			return
		}
		for _, strategyChoice := range wires {
			encoded, err := strategyChoice.Wire()
			if err != nil {
				continue
			}
			dataset = append(dataset, encoded...)
		}
		msg := Message{
			Dataset: dataset,
		}
		a.sendMessage(msg)
	case "versions":
		availableVersions, ok := fw.StrategyVersions[commands.Strategy]
		var msg Message
		if !ok {
			msg = Message{
				Valid: ok,
			}

		} else {
			msg = Message{
				Valid:    ok,
				Versions: availableVersions,
			}
		}
		a.sendMessage(msg)
	case "info":
		status := mgmt.CsStatus{
			Flags: mgmt.CsFlagEnableAdmit | mgmt.CsFlagEnableServe,
		}
		for threadID := 0; threadID < fw.NumFwThreads; threadID++ {
			thread := dispatch.GetFWThread(threadID)
			status.NCsEntries += uint64(thread.GetNumCsEntries())
		}
		// TODO fill other fields

		wire, err := status.Encode()
		if err != nil {
			return
		}
		dataset, _ := wire.Wire()
		msg := Message{
			Dataset: dataset,
		}
		a.sendMessage(msg)
	case "insert":
		hard, _ := enc.NameFromStr(commands.Name)
		table.FibStrategyTable.ClearNextHopsEnc(&hard)
		faceID := commands.FaceID
		cost := commands.Cost
		table.FibStrategyTable.InsertNextHopEnc(&hard, faceID, cost)
		a.sendMessage(MakeAck())
	case "remove":
		hard, _ := enc.NameFromStr(commands.Name)
		faceID := commands.FaceID
		table.FibStrategyTable.RemoveNextHopEnc(&hard, faceID)
		a.sendMessage(MakeAck())
	case "clear":
		hard, _ := enc.NameFromStr(commands.Name)
		table.FibStrategyTable.ClearNextHopsEnc(&hard)
		a.sendMessage(MakeAck())
	case "set":
		cap := commands.Capacity
		table.SetCsCapacity(cap)
		a.sendMessage(MakeAck())
	case "setstrategy":
		a.SetStrategy(&commands.ControlParams)
	case "unsetstrategy":
		paramName, _ := enc.NameFromStr(commands.ParamName)
		table.FibStrategyTable.UnSetStrategyEnc(&paramName)
		a.sendMessage(MakeAck())
	case "createface":
		params := commands.ControlParams
		a.CreateFace(&params)
	case "updateface":
		params := commands.ControlParams
		a.UpdateFace(&params, commands.FaceID)
	case "destroyface":
		a.RemoveFace(commands.FaceID)
	case "listface":
		a.ListFace()
	case "channels":
		a.Channels()
	case "query":
		a.Query(commands.FaceQueryFilter)
	default:
		//response = "NACK"
	}
}
func (a *AckConn) SendFace(face uint64) {
	msg := Message{
		Command: "clean",
		FaceID:  face,
	}
	a.sendMessage(msg)
}
func (a *AckConn) sendMessage(msg Message) {
	b, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("error:", err)
	}
	a.conn.Write(b)
}

func (a *AckConn) fillFaceProperties(params *mgmt.ControlParameters, selectedFace face.LinkService) {
	params.FaceID = new(uint64)
	*params.FaceID = uint64(selectedFace.FaceID())
	params.URI = selectedFace.RemoteURI()
	params.LocalURI = selectedFace.LocalURI()
	params.FacePersistency = new(uint64)
	*params.FacePersistency = uint64(selectedFace.Persistency())
	params.MTU = new(uint64)
	*params.MTU = uint64(selectedFace.MTU())

	params.Flags = new(uint64)
	*params.Flags = 0
	linkService, ok := selectedFace.(*face.NDNLPLinkService)
	if ok {
		options := linkService.Options()

		params.BaseCongestionMarkingInterval = new(uint64)
		*params.BaseCongestionMarkingInterval = uint64(options.BaseCongestionMarkingInterval.Nanoseconds())
		params.DefaultCongestionThreshold = new(uint64)
		*params.DefaultCongestionThreshold = options.DefaultCongestionThresholdBytes
		*params.Flags = options.Flags()
	}
}
func (a *AckConn) CreateFace(params *mgmt.ControlParameters) {
	var response *mgmt.ControlResponse

	// Ensure does not conflict with existing face
	existingFace := face.FaceTable.GetByURI(params.URI)
	//existingFace := mgmtconn.Acksconn.CreateFace()

	if existingFace != nil {
		responseParams := mgmt.MakeControlParameters()
		a.fillFaceProperties(responseParams, existingFace)
		responseParamsWire, err := responseParams.Encode()
		if err != nil {
			response = mgmt.MakeControlResponse(500, "Internal error", nil)
		} else {
			response = mgmt.MakeControlResponse(409, "Conflicts with existing face", responseParamsWire)
		}
		a.sendMessage(Message{ControlResponse: *response})
		return
	}

	var linkService *face.NDNLPLinkService

	if params.URI.Scheme() == "udp4" || params.URI.Scheme() == "udp6" {
		// Check that remote endpoint is not a unicast address
		if remoteAddr := net.ParseIP(params.URI.Path()); remoteAddr != nil && !remoteAddr.IsGlobalUnicast() && !remoteAddr.IsLinkLocalUnicast() {
			response = mgmt.MakeControlResponse(406, "URI must be unicast", nil)
			a.sendMessage(Message{ControlResponse: *response})
			return
		}

		// Validate and populate missing optional params
		// TODO: Validate and use LocalURI if present
		/*if params.LocalURI != nil {
			if params.LocalURI.Canonize() != nil {
				core.LogWarn(f, "Cannot canonize local URI in ControlParameters for ", interest.Name())
				response = mgmt.MakeControlResponse(406, "LocalURI could not be canonized", nil)
				return
			}
			if params.LocalURI.Scheme() != params.URI.Scheme() {
				core.LogWarn(f, "Local URI scheme does not match remote URI scheme in ControlParameters for ", interest.Name())
				response = mgmt.MakeControlResponse(406, "LocalURI scheme does not match URI scheme", nil)
				f.manager.sendResponse(response, interest, pitToken, inFace)
				return
			}
			// TODO: Check if matches a local interface IP
		}*/

		persistency := face.PersistencyPersistent
		if params.FacePersistency != nil && (*params.FacePersistency == uint64(face.PersistencyPersistent) || *params.FacePersistency == uint64(face.PersistencyPermanent)) {
			persistency = face.Persistency(*params.FacePersistency)
		} else if params.FacePersistency != nil {
			response = mgmt.MakeControlResponse(406, "Unacceptable persistency", nil)
			a.sendMessage(Message{ControlResponse: *response})
			return
		}

		baseCongestionMarkingInterval := 100 * time.Millisecond
		if params.BaseCongestionMarkingInterval != nil {
			baseCongestionMarkingInterval = time.Duration(*params.BaseCongestionMarkingInterval) * time.Nanosecond
		}

		defaultCongestionThresholdBytes := uint64(math.Pow(2, 16))
		if params.DefaultCongestionThreshold != nil {
			defaultCongestionThresholdBytes = *params.DefaultCongestionThreshold
		}

		// Create new UDP face
		transport, err := face.MakeUnicastUDPTransport(params.URI, nil, persistency)
		if err != nil {
			response = mgmt.MakeControlResponse(406, "Transport error", nil)
			a.sendMessage(Message{ControlResponse: *response})
			return
		}

		if params.MTU != nil {
			mtu := int(*params.MTU)
			if *params.MTU > tlv.MaxNDNPacketSize {
				mtu = tlv.MaxNDNPacketSize
			}
			transport.SetMTU(mtu)
		}

		// NDNLP link service parameters
		options := face.MakeNDNLPLinkServiceOptions()
		if params.Flags != nil {
			// Mask already guaranteed to be present if Flags is above
			flags := *params.Flags
			mask := *params.Mask

			if mask&face.FaceFlagLocalFields > 0 {
				// LocalFieldsEnabled
				if flags&face.FaceFlagLocalFields > 0 {
					options.IsConsumerControlledForwardingEnabled = true
					options.IsIncomingFaceIndicationEnabled = true
					options.IsLocalCachePolicyEnabled = true
				} else {
					options.IsConsumerControlledForwardingEnabled = false
					options.IsIncomingFaceIndicationEnabled = false
					options.IsLocalCachePolicyEnabled = false
				}
			}

			if mask&face.FaceFlagLpReliabilityEnabled > 0 {
				// LpReliabilityEnabled
				options.IsReliabilityEnabled = flags&face.FaceFlagLpReliabilityEnabled > 0
			}

			// Congestion control
			if mask&face.FaceFlagCongestionMarking > 0 {
				// CongestionMarkingEnabled
				options.IsCongestionMarkingEnabled = flags&face.FaceFlagCongestionMarking > 0
			}
			options.BaseCongestionMarkingInterval = baseCongestionMarkingInterval
			options.DefaultCongestionThresholdBytes = defaultCongestionThresholdBytes
		}

		linkService = face.MakeNDNLPLinkService(transport, options)
		face.FaceTable.Add(linkService)

		// Start new face
		go linkService.Run(nil)
	} else if params.URI.Scheme() == "tcp4" || params.URI.Scheme() == "tcp6" {
		// Check that remote endpoint is not a unicast address
		if remoteAddr := net.ParseIP(params.URI.Path()); remoteAddr != nil && !remoteAddr.IsGlobalUnicast() && !remoteAddr.IsLinkLocalUnicast() {

			response = mgmt.MakeControlResponse(406, "URI must be unicast", nil)
			a.sendMessage(Message{ControlResponse: *response})
			return
		}

		persistency := face.PersistencyPersistent
		if params.FacePersistency != nil && (*params.FacePersistency == uint64(face.PersistencyPersistent) || *params.FacePersistency == uint64(face.PersistencyPermanent)) {
			persistency = face.Persistency(*params.FacePersistency)
		} else if params.FacePersistency != nil {
			response = mgmt.MakeControlResponse(406, "Unacceptable persistency", nil)
			a.sendMessage(Message{ControlResponse: *response})
			return
		}

		baseCongestionMarkingInterval := 100 * time.Millisecond
		if params.BaseCongestionMarkingInterval != nil {
			baseCongestionMarkingInterval = time.Duration(*params.BaseCongestionMarkingInterval) * time.Nanosecond
		}

		defaultCongestionThresholdBytes := uint64(math.Pow(2, 16))
		if params.DefaultCongestionThreshold != nil {
			defaultCongestionThresholdBytes = *params.DefaultCongestionThreshold
		}

		// Create new TCP face
		transport, err := face.MakeUnicastTCPTransport(params.URI, nil, persistency)
		if err != nil {
			response = mgmt.MakeControlResponse(406, "Transport error", nil)
			a.sendMessage(Message{ControlResponse: *response})
			return
		}

		if params.MTU != nil {
			mtu := int(*params.MTU)
			if *params.MTU > tlv.MaxNDNPacketSize {
				mtu = tlv.MaxNDNPacketSize
			}
			transport.SetMTU(mtu)
		}

		// NDNLP link service parameters
		options := face.MakeNDNLPLinkServiceOptions()
		if params.Flags != nil {
			// Mask already guaranteed to be present if Flags is above
			flags := *params.Flags
			mask := *params.Mask

			if mask&face.FaceFlagLocalFields > 0 {
				// LocalFieldsEnabled
				if flags&face.FaceFlagLocalFields > 0 {
					options.IsConsumerControlledForwardingEnabled = true
					options.IsIncomingFaceIndicationEnabled = true
					options.IsLocalCachePolicyEnabled = true
				} else {
					options.IsConsumerControlledForwardingEnabled = false
					options.IsIncomingFaceIndicationEnabled = false
					options.IsLocalCachePolicyEnabled = false
				}
			}

			if mask&face.FaceFlagLpReliabilityEnabled > 0 {
				// LpReliabilityEnabled
				options.IsReliabilityEnabled = flags&face.FaceFlagLpReliabilityEnabled > 0
			}

			// Congestion control
			if mask&face.FaceFlagCongestionMarking > 0 {
				// CongestionMarkingEnabled
				options.IsCongestionMarkingEnabled = flags&face.FaceFlagCongestionMarking > 0
			}
			options.BaseCongestionMarkingInterval = baseCongestionMarkingInterval
			options.DefaultCongestionThresholdBytes = defaultCongestionThresholdBytes
		}

		linkService = face.MakeNDNLPLinkService(transport, options)
		face.FaceTable.Add(linkService)

		// Start new face
		go linkService.Run(nil)
	} else {
		// Unsupported scheme
		response = mgmt.MakeControlResponse(406, "Unsupported scheme "+params.URI.Scheme(), nil)
		a.sendMessage(Message{ControlResponse: *response})
		return
	}

	if linkService == nil {
		// Internal failure --> 504
		response = mgmt.MakeControlResponse(504, "Transport error when creating face", nil)
		a.sendMessage(Message{ControlResponse: *response})
		return
	}

	responseParams := mgmt.MakeControlParameters()
	a.fillFaceProperties(responseParams, linkService)
	responseParamsWire, err := responseParams.Encode()
	if err != nil {
		response = mgmt.MakeControlResponse(500, "Internal error", nil)
	} else {
		response = mgmt.MakeControlResponse(200, "OK", responseParamsWire)
	}
	a.sendMessage(Message{ControlResponse: *response})
}
func (a *AckConn) UpdateFace(params *mgmt.ControlParameters, faceId uint64) {
	var response *mgmt.ControlResponse
	faceID := faceId
	if params.FaceID != nil && *params.FaceID != 0 {
		faceID = *params.FaceID
	}

	// Validate parameters

	responseParams := mgmt.MakeControlParameters()
	areParamsValid := true

	selectedFace := face.FaceTable.Get(faceID)
	if selectedFace == nil {
		responseParams.FaceID = new(uint64)
		*responseParams.FaceID = faceID
		responseParamsWire, err := responseParams.Encode()
		if err != nil {
			response = mgmt.MakeControlResponse(500, "Internal error", nil)
		} else {
			response = mgmt.MakeControlResponse(404, "Face does not exist", responseParamsWire)
		}
		a.sendMessage(Message{ControlResponse: *response})
		return
	}

	// Can't update null (or internal) faces via management
	if selectedFace.RemoteURI().Scheme() == "null" || selectedFace.RemoteURI().Scheme() == "internal" {
		responseParams.FaceID = new(uint64)
		*responseParams.FaceID = faceID
		responseParamsWire, err := responseParams.Encode()
		if err != nil {
			response = mgmt.MakeControlResponse(500, "Internal error", nil)
		} else {
			response = mgmt.MakeControlResponse(401, "Face cannot be updated via management", responseParamsWire)
		}
		a.sendMessage(Message{ControlResponse: *response})
		return
	}

	if params.FacePersistency != nil {
		if selectedFace.RemoteURI().Scheme() == "ether" && *params.FacePersistency != uint64(face.PersistencyPermanent) {
			responseParams.FacePersistency = new(uint64)
			*responseParams.FacePersistency = *params.FacePersistency
			areParamsValid = false
		} else if (selectedFace.RemoteURI().Scheme() == "udp4" || selectedFace.RemoteURI().Scheme() == "udp6") && *params.FacePersistency != uint64(face.PersistencyPersistent) && *params.FacePersistency != uint64(face.PersistencyPermanent) {
			responseParams.FacePersistency = new(uint64)
			*responseParams.FacePersistency = *params.FacePersistency
			areParamsValid = false
		} else if selectedFace.LocalURI().Scheme() == "unix" && *params.FacePersistency != uint64(face.PersistencyPersistent) {
			responseParams.FacePersistency = new(uint64)
			*responseParams.FacePersistency = *params.FacePersistency
			areParamsValid = false
		}
	}

	if (params.Flags != nil && params.Mask == nil) || (params.Flags == nil && params.Mask != nil) {
		if params.Flags != nil {
			responseParams.Flags = new(uint64)
			*responseParams.Flags = *params.Flags
		}
		if params.Mask != nil {
			responseParams.Mask = new(uint64)
			*responseParams.Mask = *params.Mask
		}
		areParamsValid = false
	}

	if !areParamsValid {
		response = mgmt.MakeControlResponse(409, "ControlParameters are incorrect", nil)
		a.sendMessage(Message{ControlResponse: *response})
		return
	}

	// Actually perform face updates
	// Persistency
	if params.FacePersistency != nil {
		// Correctness of FacePersistency already validated
		selectedFace.SetPersistency(face.Persistency(*params.FacePersistency))
	}

	options := selectedFace.(*face.NDNLPLinkService).Options()

	// Congestion
	if params.BaseCongestionMarkingInterval != nil && time.Duration(*params.BaseCongestionMarkingInterval)*time.Nanosecond != options.BaseCongestionMarkingInterval {
		options.BaseCongestionMarkingInterval = time.Duration(*params.BaseCongestionMarkingInterval) * time.Nanosecond
	}

	if params.DefaultCongestionThreshold != nil && *params.DefaultCongestionThreshold != options.DefaultCongestionThresholdBytes {
		options.DefaultCongestionThresholdBytes = *params.DefaultCongestionThreshold
	}

	// MTU
	if params.MTU != nil {
		newMTU := int(*params.MTU)
		if *params.MTU > tlv.MaxNDNPacketSize {
			newMTU = tlv.MaxNDNPacketSize
		}
		selectedFace.SetMTU(newMTU)
	}

	// Flags
	if params.Flags != nil {
		// Presence of mask already validated
		flags := *params.Flags
		mask := *params.Mask

		if mask&face.FaceFlagLocalFields > 0 {
			// Update LocalFieldsEnabled
			if flags&face.FaceFlagLocalFields > 0 {
				options.IsConsumerControlledForwardingEnabled = true
				options.IsIncomingFaceIndicationEnabled = true
				options.IsLocalCachePolicyEnabled = true
			} else {
				options.IsConsumerControlledForwardingEnabled = false
				options.IsIncomingFaceIndicationEnabled = false
				options.IsLocalCachePolicyEnabled = false
			}
		}

		if mask&face.FaceFlagLpReliabilityEnabled > 0 {
			// Update LpReliabilityEnabled
			options.IsReliabilityEnabled = flags&face.FaceFlagLpReliabilityEnabled > 0
		}

		if mask&face.FaceFlagCongestionMarking > 0 {
			// Update CongestionMarkingEnabled
			options.IsCongestionMarkingEnabled = flags&face.FaceFlagCongestionMarking > 0
		}
	}

	selectedFace.(*face.NDNLPLinkService).SetOptions(options)

	a.fillFaceProperties(responseParams, selectedFace)
	responseParams.URI = nil
	responseParams.LocalURI = nil
	responseParamsWire, err := responseParams.Encode()
	if err != nil {
		response = mgmt.MakeControlResponse(500, "Internal error", nil)
	} else {
		response = mgmt.MakeControlResponse(200, "OK", responseParamsWire)
	}
	a.sendMessage(Message{ControlResponse: *response})
}

func (a *AckConn) RemoveFace(faceID uint64) {
	if face.FaceTable.Get(faceID) != nil {
		face.FaceTable.Remove(faceID)
	}
	msg := MakeAck()
	a.sendMessage(msg)
}

func (a *AckConn) ListFace() {
	faces := make(map[uint64]face.LinkService)
	faceIDs := make([]uint64, 0)
	for _, face := range face.FaceTable.GetAll() {
		faces[face.FaceID()] = face
		faceIDs = append(faceIDs, face.FaceID())
	}
	// We have to sort these or they appear in a strange order
	sort.Slice(faceIDs, func(a int, b int) bool { return faceIDs[a] < faceIDs[b] })
	dataset := make([]byte, 0)
	for _, faceID := range faceIDs {
		dataset = append(dataset, a.createDataset(faces[faceID])...)
	}
	msg := Message{
		Dataset: dataset,
	}
	a.sendMessage(msg)
}

func (a *AckConn) createDataset(selectedFace face.LinkService) []byte {
	faceDataset := mgmt.MakeFaceStatus()
	faceDataset.FaceID = uint64(selectedFace.FaceID())
	faceDataset.URI = selectedFace.RemoteURI()
	faceDataset.LocalURI = selectedFace.LocalURI()
	if selectedFace.ExpirationPeriod() != 0 {
		faceDataset.ExpirationPeriod = new(uint64)
		*faceDataset.ExpirationPeriod = uint64(selectedFace.ExpirationPeriod().Milliseconds())
	}
	faceDataset.FaceScope = uint64(selectedFace.Scope())
	faceDataset.FacePersistency = uint64(selectedFace.Persistency())
	faceDataset.LinkType = uint64(selectedFace.LinkType())
	faceDataset.MTU = new(uint64)
	*faceDataset.MTU = uint64(selectedFace.MTU())
	faceDataset.NInInterests = selectedFace.NInInterests()
	faceDataset.NInData = selectedFace.NInData()
	faceDataset.NInNacks = 0
	faceDataset.NOutInterests = selectedFace.NOutInterests()
	faceDataset.NOutData = selectedFace.NOutData()
	faceDataset.NOutNacks = 0
	faceDataset.NInBytes = selectedFace.NInBytes()
	faceDataset.NOutBytes = selectedFace.NOutBytes()
	linkService, ok := selectedFace.(*face.NDNLPLinkService)
	if ok {
		options := linkService.Options()

		faceDataset.BaseCongestionMarkingInterval = new(uint64)
		*faceDataset.BaseCongestionMarkingInterval = uint64(options.BaseCongestionMarkingInterval.Nanoseconds())
		faceDataset.DefaultCongestionThreshold = new(uint64)
		*faceDataset.DefaultCongestionThreshold = options.DefaultCongestionThresholdBytes
		faceDataset.Flags = options.Flags()
		if options.IsConsumerControlledForwardingEnabled {
			// This one will only be enabled if the other two local fields are enabled (and vice versa)
			faceDataset.Flags |= face.FaceFlagLocalFields
		}
		if options.IsReliabilityEnabled {
			faceDataset.Flags |= face.FaceFlagLpReliabilityEnabled
		}
		if options.IsCongestionMarkingEnabled {
			faceDataset.Flags |= face.FaceFlagCongestionMarking
		}
	}

	faceDatasetEncoded, err := faceDataset.Encode()
	if err != nil {
		return []byte{}
	}
	faceDatasetWire, err := faceDatasetEncoded.Wire()
	if err != nil {
		return []byte{}
	}
	return faceDatasetWire
}

func (a *AckConn) Channels() {
	dataset := make([]byte, 0)
	// UDP channel
	ifaces, err := net.Interfaces()
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return
		}
		for _, addr := range addrs {
			ipAddr := addr.(*net.IPNet)

			ipVersion := 4
			path := ipAddr.IP.String()
			if ipAddr.IP.To4() == nil {
				ipVersion = 6
				path += "%" + iface.Name
			}

			if !addr.(*net.IPNet).IP.IsLoopback() {
				uri := ndn.MakeUDPFaceURI(ipVersion, path, face.UDPUnicastPort)
				channel := mgmt.MakeChannelStatus(uri)
				channelEncoded, err := channel.Encode()
				if err != nil {
					continue
				}
				channelWire, err := channelEncoded.Wire()
				if err != nil {
					continue
				}
				dataset = append(dataset, channelWire...)
			}
		}
	}

	// Unix channel
	uri := ndn.MakeUnixFaceURI(face.UnixSocketPath)
	channel := mgmt.MakeChannelStatus(uri)
	channelEncoded, err := channel.Encode()
	if err != nil {
		return
	}
	channelWire, err := channelEncoded.Wire()
	if err != nil {
		return
	}
	dataset = append(dataset, channelWire...)
	msg := Message{
		Dataset: dataset,
	}
	a.sendMessage(msg)
}
func (a *AckConn) Query(filter mgmt.FaceQueryFilter) {
	faces := face.FaceTable.GetAll()
	matchingFaces := make([]int, 0)
	for pos, face := range faces {
		if filter.FaceID != nil && *filter.FaceID != face.FaceID() {
			continue
		}

		if filter.URIScheme != nil && *filter.URIScheme != face.LocalURI().Scheme() && *filter.URIScheme != face.RemoteURI().Scheme() {
			continue
		}

		if filter.URI != nil && filter.URI.String() != face.RemoteURI().String() {
			continue
		}

		if filter.LocalURI != nil && filter.LocalURI.String() != face.LocalURI().String() {
			continue
		}

		if filter.FaceScope != nil && *filter.FaceScope != uint64(face.Scope()) {
			continue
		}

		if filter.FacePersistency != nil && *filter.FacePersistency != uint64(face.Persistency()) {
			continue
		}

		if filter.LinkType != nil && *filter.LinkType != uint64(face.LinkType()) {
			continue
		}

		matchingFaces = append(matchingFaces, pos)
	}

	// We have to sort these or they appear in a strange order
	//sort.Slice(matchingFaces, func(a int, b int) bool { return matchingFaces[a] < matchingFaces[b] })

	dataset := make([]byte, 0)
	for _, pos := range matchingFaces {
		dataset = append(dataset, a.createDataset(faces[pos])...)
	}
	msg := Message{
		Dataset: dataset,
	}
	a.sendMessage(msg)
}

func (a *AckConn) SetStrategy(params *mgmt.ControlParameters) {

}
