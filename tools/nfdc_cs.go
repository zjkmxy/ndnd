package tools

import (
	"fmt"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
)

func (n *Nfdc) ExecCsInfo(args []string) {
	suffix := enc.Name{
		enc.NewStringComponent(enc.TypeGenericNameComponent, "cs"),
		enc.NewStringComponent(enc.TypeGenericNameComponent, "info"),
	}

	data, err := n.fetchStatusDataset(suffix)
	if err != nil {
		log.Fatalf("Error fetching status dataset: %+v", err)
		return
	}

	status, err := mgmt.ParseCsInfoMsg(enc.NewWireReader(data), true)
	if err != nil || status.CsInfo == nil {
		log.Fatalf("Error parsing CS info: %+v", err)
		return
	}

	info := status.CsInfo

	fmt.Println("CS information:")
	n.statusPadding = 10
	n.printStatusLine("capacity", info.Capacity)
	n.printStatusLine("admit", info.Flags&mgmt.CsEnableAdmit != 0)
	n.printStatusLine("serve", info.Flags&mgmt.CsEnableServe != 0)
	n.printStatusLine("nEntries", info.NCsEntries)
	n.printStatusLine("nHits", info.NHits)
	n.printStatusLine("nMisses", info.NMisses)
}
