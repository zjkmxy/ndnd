package tools

import (
	"github.com/named-data/ndnd/std/log"
	mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
)

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
