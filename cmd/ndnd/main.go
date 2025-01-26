package main

import (
	"fmt"
	"os"

	"github.com/named-data/ndnd/cmd"
	dv "github.com/named-data/ndnd/dv/executor"
	fw "github.com/named-data/ndnd/fw/executor"
	"github.com/named-data/ndnd/std/utils"
	"github.com/named-data/ndnd/std/utils/toolutils"
	"github.com/named-data/ndnd/tools"
	"github.com/named-data/ndnd/tools/dvc"
	"github.com/named-data/ndnd/tools/nfdc"
	"github.com/named-data/ndnd/tools/sec"
)

func main() {
	tree := toolutils.CmdTree{
		Name: "ndnd",
		Help: fmt.Sprintf("Named Data Networking Daemon (%s)", utils.NDNdVersion),
		Sub: []*toolutils.CmdTree{{
			Name: "fw",
			Help: "NDN Forwarding Daemon",
			Sub: append([]*toolutils.CmdTree{{
				Name: "run",
				Help: "Start the NDN Forwarding Daemon",
				Fun:  fw.Main,
			}, {},
			}, nfdc.Tree().Sub...),
		}, {
			Name: "dv",
			Help: "NDN Distance Vector Routing Daemon",
			Sub: append([]*toolutils.CmdTree{{
				Name: "run",
				Help: "Start the NDN Distance Vector Routing Daemon",
				Fun:  dv.Main,
			}, {},
			}, dvc.Tree().Sub...),
		}, {
			Name: "daemon",
			Help: "NDN Combined Daemon",
			Fun:  cmd.Daemon,
		}, {
			// tools separator
		}, {
			Name: "sec",
			Help: "NDN Security Utilities",
			Sub:  sec.Tree().Sub,
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

	// Handle top level flags
	if len(args) >= 2 && args[1] == "-version" {
		fmt.Println(utils.NDNdVersion)
		os.Exit(0)
	}

	tree.Execute(args)
}
