package tools

import (
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/engine"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/utils"
)

func GetNfdcCmdTree() utils.CmdTree {
	nfdc := &Nfdc{}
	execCmd := func(mod string, cmd string) func([]string) {
		return func(args []string) {
			nfdc.ExecCmd(mod, cmd, args)
		}
	}

	return utils.CmdTree{
		Name: "nfdc",
		Help: "NDNd Forwarder Control",
		Sub: []*utils.CmdTree{{
			Name: "status",
			Help: "Print general status",
			Fun:  nfdc.ExecStatusGeneral,
		}, {
			Name: "face list",
			Help: "Print face table",
			Fun:  nfdc.ExecFaceList,
		}, {
			Name: "route list",
			Help: "Print RIB routes",
			Fun:  nfdc.ExecRouteList,
		}, {
			Name: "route add",
			Help: "Add a route to the RIB",
			Fun:  execCmd("rib", "add"),
		}, {
			Name: "fib list",
			Help: "Print FIB entries",
			Fun:  nfdc.ExecFibList,
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
