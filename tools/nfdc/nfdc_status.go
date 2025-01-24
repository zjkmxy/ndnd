package nfdc

import (
	"fmt"
	"os"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
	"github.com/named-data/ndnd/std/utils"
)

func (n *Nfdc) ExecStatusGeneral(args []string) {
	suffix := enc.Name{
		enc.NewStringComponent(enc.TypeGenericNameComponent, "status"),
		enc.NewStringComponent(enc.TypeGenericNameComponent, "general"),
	}

	data, err := n.fetchStatusDataset(suffix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching status dataset: %+v\n", err)
		os.Exit(1)
		return
	}

	status, err := mgmt.ParseGeneralStatus(enc.NewBufferReader(data), true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing general status: %+v\n", err)
		os.Exit(1)
		return
	}

	p := utils.StatusPrinter{File: os.Stdout, Padding: 24}
	fmt.Println("General NFD status:")
	p.Print("version", status.NfdVersion)
	p.Print("startTime", time.Unix(0, int64(status.StartTimestamp)))
	p.Print("currentTime", time.Unix(0, int64(status.CurrentTimestamp)))
	p.Print("uptime", (status.CurrentTimestamp - status.StartTimestamp))
	p.Print("nNameTreeEntries", status.NNameTreeEntries)
	p.Print("nFibEntries", status.NFibEntries)
	p.Print("nPitEntries", status.NCsEntries)
	p.Print("nMeasurementsEntries", status.NMeasurementsEntries)
	p.Print("nCsEntries", status.NCsEntries)
	p.Print("nInInterests", status.NInInterests)
	p.Print("nOutInterests", status.NOutInterests)
	p.Print("nInData", status.NInData)
	p.Print("nOutData", status.NOutData)
	p.Print("nInNacks", status.NInNacks)
	p.Print("nOutNacks", status.NOutNacks)
	p.Print("nSatisfiedInterests", status.NSatisfiedInterests)
	p.Print("nUnsatisfiedInterests", status.NUnsatisfiedInterests)
}
