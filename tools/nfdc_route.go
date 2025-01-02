package tools

import (
	"fmt"
	"strings"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
)

func (n *Nfdc) ExecRouteList(args []string) {
	suffix := enc.Name{
		enc.NewStringComponent(enc.TypeGenericNameComponent, "rib"),
		enc.NewStringComponent(enc.TypeGenericNameComponent, "list"),
	}

	data, err := n.fetchStatusDataset(suffix)
	if err != nil {
		log.Fatalf("Error fetching status dataset: %+v", err)
		return
	}

	status, err := mgmt.ParseRibStatus(enc.NewWireReader(data), true)
	if err != nil {
		log.Fatalf("Error parsing RIB status: %+v", err)
		return
	}

	for _, entry := range status.Entries {
		for _, route := range entry.Routes {
			expiry := "never"
			if route.ExpirationPeriod != nil {
				expiry = (time.Duration(*route.ExpirationPeriod) * time.Millisecond).String()
			}

			// TODO: convert origin, flags to string
			fmt.Printf("prefix=%s nexthop=%d origin=%d cost=%d flags=%d expires=%s\n",
				entry.Name, route.FaceId, route.Origin, route.Cost, route.Flags, expiry)
		}
	}
}

func (n *Nfdc) ExecFibList(args []string) {
	suffix := enc.Name{
		enc.NewStringComponent(enc.TypeGenericNameComponent, "fib"),
		enc.NewStringComponent(enc.TypeGenericNameComponent, "list"),
	}

	data, err := n.fetchStatusDataset(suffix)
	if err != nil {
		log.Fatalf("Error fetching status dataset: %+v", err)
		return
	}

	status, err := mgmt.ParseFibStatus(enc.NewWireReader(data), true)
	if err != nil {
		log.Fatalf("Error parsing RIB status: %+v", err)
		return
	}

	fmt.Println("FIB:")
	for _, entry := range status.Entries {
		nexthops := make([]string, 0)
		for _, record := range entry.NextHopRecords {
			nexthops = append(nexthops, fmt.Sprintf("faceid=%d (cost=%d)", record.FaceId, record.Cost))
		}
		fmt.Printf("  %s nexthops={%s}\n", entry.Name, strings.Join(nexthops, ", "))
	}
}
