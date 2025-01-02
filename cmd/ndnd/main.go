package main

import (
	"os"

	dv "github.com/named-data/ndnd/dv/executor"
	fw "github.com/named-data/ndnd/fw/executor"
	"github.com/named-data/ndnd/std/utils"
	tools "github.com/named-data/ndnd/tools"
	nfdc "github.com/named-data/ndnd/tools/nfdc"
)

func main() {
	// create a command tree
	tree := utils.CmdTree{
		Name: "ndnd",
		Help: "Named Data Networking Daemon",
		Sub: []*utils.CmdTree{{
			Name: "fw",
			Help: "NDN Forwarding Daemon",
			Sub:  fwSubtree(),
		}, {
			Name: "dv",
			Help: "NDN Distance Vector Routing Daemon",
			Sub: []*utils.CmdTree{{
				Name: "run",
				Help: "Start the NDN Distance Vector Routing Daemon",
				Fun:  dv.Main,
			}},
		}, {
			// tools separator
		}, {
			Name: "ping",
			Help: "Send Interests to an NDN ping server",
			Fun:  tools.RunPingClient,
		}, {
			Name: "pingserver",
			Help: "Start an NDN ping server under a prefix",
			Fun:  tools.RunPingServer,
		}, {
			Name: "cat",
			Help: "Retrieve data under a prefix",
			Fun:  tools.RunCatChunks,
		}, {
			Name: "put",
			Help: "Publish data under a prefix",
			Fun:  tools.RunPutChunks,
		}},
	}

	// Parse the command line arguments
	args := os.Args
	args[0] = tree.Name
	tree.Execute(args)
}

func fwSubtree() []*utils.CmdTree {
	tree := []*utils.CmdTree{{
		Name: "run",
		Help: "Start the NDN Forwarding Daemon",
		Fun:  fw.Main,
	}, {
		// separator for nfdc
	}}
	tree = append(tree, nfdc.GetNfdcCmdTree().Sub...)
	return tree
}
