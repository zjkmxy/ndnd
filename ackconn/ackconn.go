package ackconn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/dispatch"
	"github.com/named-data/YaNFD/face"
	"github.com/named-data/YaNFD/fw"
	"github.com/named-data/YaNFD/ndn/mgmt"
	"github.com/named-data/YaNFD/table"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

var AckChannel AckConn

type AckConn struct {
	conn       net.Conn
	socketFile string
}
type Message struct {
	Command   string   `json:"command"`
	Name      string   `json:"name"`
	ParamName string   `json:"paramname"`
	FaceID    uint64   `json:"faceid"`
	Cost      uint64   `json:"cost"`
	Strategy  string   `json:"strategy"`
	Capacity  int      `json:"capacity"`
	Versions  []uint64 `json:"versions"`
	Dataset   []byte   `json:"dataset"`
	Valid     bool     `json:"valid"`
}

func MakeAck() Message {
	msg := Message{
		Valid: true,
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
		paramName, _ := enc.NameFromStr(commands.ParamName)
		strategy, _ := enc.NameFromStr(commands.Strategy)
		table.FibStrategyTable.SetStrategyEnc(&paramName, &strategy)
		a.sendMessage(MakeAck())
	case "unsetstrategy":
		paramName, _ := enc.NameFromStr(commands.ParamName)
		table.FibStrategyTable.UnSetStrategyEnc(&paramName)
		a.sendMessage(MakeAck())
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
