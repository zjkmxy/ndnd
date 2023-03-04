package mgmtconn

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/named-data/YaNFD/table"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

type MgmtConn struct {
	conn       net.Conn
	socketFile string
}

var Channel MgmtConn

func (m *MgmtConn) Make(socketFile string) {
	m.socketFile = socketFile
}

func (m *MgmtConn) Send(face uint64) {
	msg := fmt.Sprintf("%d", face)
	m.conn.Write([]byte(msg))
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
	fibcommand := string(buf)
	fmt.Println(fibcommand)
	command := strings.Split(fibcommand, ",")
	switch command[0] {
	case "insert":
		hard, _ := enc.NameFromStr(command[1])
		table.FibStrategyTable.ClearNextHops1(&hard)
		faceID, _ := strconv.Atoi(command[2])
		cost, _ := strconv.Atoi(command[3])
		table.FibStrategyTable.InsertNextHop1(&hard, uint64(faceID), uint64(cost))
		// log := fmt.Sprintf("inserted %s, %s, %s", command[1], command[2], command[3])
		// fmt.Println(log)
	case "remove":
		hard, _ := enc.NameFromStr(command[1])
		faceID, _ := strconv.Atoi(command[2])
		table.FibStrategyTable.RemoveNextHop1(&hard, uint64(faceID))
		// log := fmt.Sprintf("removed %s, %s", command[1], command[2])
		// fmt.Println(log)
	case "clear":
		hard, _ := enc.NameFromStr(command[1])
		table.FibStrategyTable.ClearNextHops1(&hard)
		// log := fmt.Sprintf("cleared %s", command[1])
		// fmt.Println(log)
	case "set":
		cap, _ := strconv.Atoi(command[1])
		table.SetCsCapacity(cap)
	case "setstrategy":
	case "unsetstrategy":
	default:
		//response = "NACK"
	}
}
