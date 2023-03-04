package ackconn

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/dispatch"
	"github.com/named-data/YaNFD/face"
	"github.com/named-data/YaNFD/fw"
	"github.com/named-data/YaNFD/ndn/mgmt"
	"github.com/named-data/YaNFD/table"
)

var AckChannel AckConn

type AckConn struct {
	conn       net.Conn
	socketFile string
}

func (a *AckConn) Make(socketFile string) {
	a.socketFile = socketFile
}

func (a *AckConn) SendFace(face uint64) {
	msg := fmt.Sprintf("%d", face)
	a.conn.Write([]byte(msg))
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
	fibcommand := string(buf)
	fmt.Println(fibcommand)
	command := strings.Split(fibcommand, ",")
	switch command[0] {
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
		a.conn.Write(dataset)
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
		a.conn.Write(dataset)
	case "faceid":
		faceID, _ := strconv.Atoi(string(command[1]))
		if face.FaceTable.Get(uint64(faceID)) != nil {
			a.conn.Write([]byte("ack"))
		} else {
			a.conn.Write([]byte("nack"))
		}
	default:
		//response = "NACK"
	}
}
