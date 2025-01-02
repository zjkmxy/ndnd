package tools

import (
	"fmt"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/engine"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
	"github.com/named-data/ndnd/std/utils"
)

func GetNfdcCmdTree() utils.CmdTree {
	nfdc := &Nfdc{}
	execCmd := func(mod string, cmd string) func([]string) {
		return func(args []string) {
			nfdc.ExecCmd(mod, cmd, args)
		}
	}

	// all subcommands MUST be two words "module", "command"
	return utils.CmdTree{
		Name: "nfdc",
		Help: "NDNd Forwarder Control",
		Sub: []*utils.CmdTree{{
			Name: "route list",
			Help: "Print RIB routes",
			Fun:  nfdc.ExecRouteList,
		}, {
			Name: "route add",
			Help: "Control RIB and FIB entries",
			Fun:  execCmd("rib", "add"),
		}},
	}
}

type Nfdc struct {
	engine ndn.Engine
}

func (n *Nfdc) Start() {
	face := engine.NewUnixFace("/var/run/nfd/nfd.sock")
	n.engine = engine.NewBasicEngine(face)

	err := n.engine.Start()
	if err != nil {
		log.Fatalf("Unable to start engine: %+v", err)
		return
	}
}

func (n *Nfdc) Stop() {
	n.engine.Stop()
}

func (n *Nfdc) GetPrefix() enc.Name {
	return enc.Name{
		enc.NewStringComponent(enc.TypeGenericNameComponent, "localhost"),
		enc.NewStringComponent(enc.TypeGenericNameComponent, "nfd"),
	}
}

func (n *Nfdc) ExecCmd(mod string, cmd string, args []string) {
	n.Start()
	defer n.Stop()

	argmap := map[string]any{}
	ctrlArgs, err := mgmt.DictToControlArgs(argmap)
	if err != nil {
		log.Fatalf("Invalid control args: %+v", err)
		return
	}

	resp, err := n.engine.ExecMgmtCmd(mod, cmd, ctrlArgs)
	if err != nil {
		log.Fatalf("Error executing command: %+v", err)
		return
	}

	log.Infof("Response: %+v", resp)
}

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

func (n *Nfdc) fetchStatusDataset(suffix enc.Name) (enc.Wire, error) {
	n.Start()
	defer n.Stop()

	// TODO: segmented fetch once supported by fw/mgmt
	name := append(n.GetPrefix(), suffix...)
	config := &ndn.InterestConfig{
		MustBeFresh: true,
		CanBePrefix: true,
		Lifetime:    utils.IdPtr(time.Second),
		Nonce:       utils.ConvertNonce(n.engine.Timer().Nonce()),
	}
	interest, err := n.engine.Spec().MakeInterest(name, config, nil, nil)
	if err != nil {
		return nil, err
	}

	ch := make(chan ndn.ExpressCallbackArgs)
	err = n.engine.Express(interest, func(args ndn.ExpressCallbackArgs) {
		ch <- args
		close(ch)
	})
	if err != nil {
		return nil, err
	}

	res := <-ch
	if res.Result != ndn.InterestResultData {
		return nil, fmt.Errorf("interest failed: %d", res.Result)
	}

	return res.Data.Content(), nil
}
