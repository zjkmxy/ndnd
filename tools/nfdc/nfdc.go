package nfdc

import (
	"fmt"
	"os"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/engine"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/utils"
)

func Tree() *utils.CmdTree {
	t := Tool{}

	cmd := func(mod string, cmd string, defaults []string) func([]string) {
		return func(args []string) {
			t.ExecCmd(mod, cmd, args, defaults)
		}
	}

	return &utils.CmdTree{
		Name: "nfdc",
		Help: "NDNd Forwarder Control",
		Sub: []*utils.CmdTree{{
			Name: "status",
			Help: "Print general status",
			Fun:  t.ExecStatusGeneral,
		}, {
			Name: "face list",
			Help: "Print face table",
			Fun:  t.ExecFaceList,
		}, {
			Name: "face create",
			Help: "Create a face",
			Fun: cmd("faces", "create", []string{
				"persistency=persistent",
			}),
		}, {
			Name: "face destroy",
			Help: "Destroy a face",
			Fun:  cmd("faces", "destroy", []string{}),
		}, {
			Name: "route list",
			Help: "Print RIB routes",
			Fun:  t.ExecRouteList,
		}, {
			Name: "route add",
			Help: "Add a route to the RIB",
			Fun: cmd("rib", "register", []string{
				"cost=0", "origin=255",
			}),
		}, {
			Name: "route remove",
			Help: "Remove a route from the RIB",
			Fun: cmd("rib", "unregister", []string{
				"origin=255",
			}),
		}, {
			Name: "fib list",
			Help: "Print FIB entries",
			Fun:  t.ExecFibList,
		}, {
			Name: "cs info",
			Help: "Print content store info",
			Fun:  t.ExecCsInfo,
		}, {
			Name: "strategy list",
			Help: "Print strategy choices",
			Fun:  t.ExecStrategyList,
		}, {
			Name: "strategy set",
			Help: "Set strategy choice",
			Fun:  cmd("strategy-choice", "set", []string{}),
		}, {
			Name: "strategy unset",
			Help: "Unset strategy choice",
			Fun:  cmd("strategy-choice", "unset", []string{}),
		}},
	}
}

type Tool struct {
	engine ndn.Engine
}

func (t *Tool) Start() {
	if t.engine != nil {
		return
	}

	t.engine = engine.NewBasicEngine(engine.NewDefaultFace())
	err := t.engine.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to start engine: %+v\n", err)
		os.Exit(1)
		return
	}
}

func (t *Tool) Stop() {
	t.engine.Stop()
}

func (t *Tool) Prefix() enc.Name {
	return enc.Name{
		enc.LOCALHOST,
		enc.NewStringComponent(enc.TypeGenericNameComponent, "nfd"),
	}
}
