package mgmtconn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/named-data/YaNFD/table"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

type MgmtConn struct {
	conn       net.Conn
	socketFile string
}

type Message struct {
	Command   string `json:"command"`
	Name      string `json:"name"`
	ParamName string `json:"paramname"`
	FaceID    uint64 `json:"faceid"`
	Cost      uint64 `json:"cost"`
	Strategy  string `json:"strategy"`
	Capacity  int    `json:"capacity"`
}

var Channel MgmtConn

func (m *MgmtConn) Make(socketFile string) {
	m.socketFile = socketFile
}

func (m *MgmtConn) Send(face uint64) {
	msg := Message{
		FaceID: face,
	}
	m.sendMessage(msg)
}

func (m *MgmtConn) RunReceive() {
	// listen to incoming unix packets
	os.Remove(m.socketFile)
	listener, err := net.Listen("unixpacket", m.socketFile)
	if err := os.Chmod(m.socketFile, 0777); err != nil {
		fmt.Println(err)
	}
	if err != nil {
		return
	}
	defer listener.Close()
	m.conn, _ = listener.Accept()
	for {
		buf := make([]byte, 8800)
		size, err := m.conn.Read(buf)
		if err != nil {
			continue
		}
		m.process(size, buf)
	}
}

func (m *MgmtConn) process(size int, buf []byte) {
	//var response string = "test"
	buf = bytes.Trim(buf, "\x00")
	var commands Message
	err := json.Unmarshal(buf, &commands)
	if err != nil {
		fmt.Println("error:", err)
	}
	switch commands.Command {
	case "insert":
		hard, _ := enc.NameFromStr(commands.Name)
		table.FibStrategyTable.ClearNextHopsEnc(&hard)
		faceID := commands.FaceID
		cost := commands.Cost
		table.FibStrategyTable.InsertNextHopEnc(&hard, faceID, cost)
	case "remove":
		hard, _ := enc.NameFromStr(commands.Name)
		faceID := commands.FaceID
		table.FibStrategyTable.RemoveNextHopEnc(&hard, faceID)
	case "clear":
		hard, _ := enc.NameFromStr(commands.Name)
		table.FibStrategyTable.ClearNextHopsEnc(&hard)
	case "set":
		cap := commands.Capacity
		table.SetCsCapacity(cap)
	case "setstrategy":
		paramName, _ := enc.NameFromStr(commands.ParamName)
		strategy, _ := enc.NameFromStr(commands.Strategy)
		table.FibStrategyTable.SetStrategyEnc(&paramName, &strategy)
	case "unsetstrategy":
		paramName, _ := enc.NameFromStr(commands.ParamName)
		table.FibStrategyTable.UnSetStrategyEnc(&paramName)
	default:
		//response = "NACK"
	}
}

func (m *MgmtConn) sendMessage(msg Message) {
	b, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("error:", err)
	}
	m.conn.Write(b)
}
