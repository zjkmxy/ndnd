package tools

import (
	"fmt"
	"strings"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
)

func (n *Nfdc) ExecStatusGeneral(args []string) {
	suffix := enc.Name{
		enc.NewStringComponent(enc.TypeGenericNameComponent, "status"),
		enc.NewStringComponent(enc.TypeGenericNameComponent, "general"),
	}

	data, err := n.fetchStatusDataset(suffix)
	if err != nil {
		log.Fatalf("Error fetching status dataset: %+v", err)
		return
	}

	status, err := mgmt.ParseGeneralStatus(enc.NewWireReader(data), true)
	if err != nil {
		log.Fatalf("Error parsing general status: %+v", err)
		return
	}

	print := func(key string, value any) {
		if _, ok := value.(uint64); ok {
			value = fmt.Sprintf("%d", value)
		}

		padding := 24 - len(key)
		fmt.Printf("%s%s=%s\n", strings.Repeat(" ", padding), key, value)
	}

	fmt.Println("General NFD status:")
	print("version", status.NfdVersion)
	print("startTime", time.UnixMilli(int64(status.StartTimestamp)))
	print("currentTime", time.UnixMilli(int64(status.CurrentTimestamp)))
	print("uptime", time.Duration(status.CurrentTimestamp-status.StartTimestamp)*time.Millisecond)
	print("nNameTreeEntries", status.NNameTreeEntries)
	print("nFibEntries", status.NFibEntries)
	print("nPitEntries", status.NCsEntries)
	print("nMeasurementsEntries", status.NMeasurementsEntries)
	print("nCsEntries", status.NCsEntries)
	print("nInInterests", status.NInInterests)
	print("nOutInterests", status.NOutInterests)
	print("nInData", status.NInData)
	print("nOutData", status.NOutData)
	print("nInNacks", status.NInNacks)
	print("nOutNacks", status.NOutNacks)
	print("nSatisfiedInterests", status.NSatisfiedInterests)
	print("nUnsatisfiedInterests", status.NUnsatisfiedInterests)
}
