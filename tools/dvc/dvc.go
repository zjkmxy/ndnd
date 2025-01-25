package dvc

import (
	"fmt"
	"os"

	"github.com/named-data/ndnd/std/engine"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/utils/toolutils"
)

func Tree() *toolutils.CmdTree {
	t := Tool{}

	return &toolutils.CmdTree{
		Name: "dvc",
		Help: "NDN Distance Vector Control",
		Sub: []*toolutils.CmdTree{{
			Name: "status",
			Help: "Get general status of the router",
			Fun:  t.RunDvStatus,
		}, {
			Name: "link create",
			Help: "Create a new active neighbor link",
			Fun:  t.RunDvLinkCreate,
		}, {
			Name: "link destroy",
			Help: "Destroy an active neighbor link",
			Fun:  t.RunDvLinkDestroy,
		}},
	}
}

type Tool struct {
	engine ndn.Engine
}

func (t *Tool) Start() {
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
