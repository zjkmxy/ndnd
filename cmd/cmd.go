package cmd

import (
	dv "github.com/named-data/ndnd/dv/cmd"
	fw "github.com/named-data/ndnd/fw/cmd"
	"github.com/named-data/ndnd/std/utils"
	"github.com/named-data/ndnd/tools"
	"github.com/named-data/ndnd/tools/dvc"
	"github.com/named-data/ndnd/tools/nfdc"
	"github.com/named-data/ndnd/tools/sec"
	"github.com/spf13/cobra"
)

const banner = `
  _   _ ____  _   _     _
 | \ | |  _ \| \ | | __| |
 |  \| | | | |  \| |/ _  |
 | |\  | |_| | |\  | (_| |
 |_| \_|____/|_| \_|\____|

Named Data Networking Daemon
`

var CmdNDNd = &cobra.Command{
	Use:     "ndnd",
	Short:   "Named Data Networking Daemon",
	Long:    banner[1:],
	Version: utils.NDNdVersion,
}

func init() {
	CmdNDNd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	CmdNDNd.PersistentFlags().Lookup("help").Hidden = true

	CmdNDNd.AddGroup(&cobra.Group{
		ID:    "daemons",
		Title: "NDN Daemons",
	})
	CmdNDNd.AddCommand(cmdFw())
	CmdNDNd.AddCommand(cmdDv())

	CmdNDNd.AddGroup(&cobra.Group{
		ID:    "sec",
		Title: "Security Tools",
	})
	CmdNDNd.AddCommand(sec.CmdSec())
	CmdNDNd.AddCommand(sec.CmdCertCli())

	CmdNDNd.AddGroup(&cobra.Group{
		ID:    "tools",
		Title: "Debug Tools",
	})
	CmdNDNd.AddCommand(tools.CmdCatChunks())
	CmdNDNd.AddCommand(tools.CmdPutChunks())
	CmdNDNd.AddCommand(tools.CmdPingClient())
	CmdNDNd.AddCommand(tools.CmdPingServer())
}

func cmdFw() *cobra.Command {
	cmdFw := &cobra.Command{
		Use:     "fw",
		Short:   "NDN Forwarding Daemon",
		GroupID: "daemons",
	}

	cmdFw.AddGroup(&cobra.Group{
		ID:    "run",
		Title: "Forwarder Daemon",
	})
	fw.CmdYaNFD.Use = "run config-file"
	fw.CmdYaNFD.Short = "Start the NDN Forwarding Daemon"
	cmdFw.AddCommand(fw.CmdYaNFD)

	cmdFw.AddGroup(&cobra.Group{
		ID:    "nfdc",
		Title: "Forwarder Control",
	})
	for _, sub := range nfdc.Cmds() {
		sub.GroupID = "nfdc"
		cmdFw.AddCommand(sub)
	}

	return cmdFw
}

func cmdDv() *cobra.Command {
	cmdDv := &cobra.Command{
		Use:     "dv",
		Short:   "NDN Distance Vector Daemon",
		GroupID: "daemons",
	}

	cmdDv.AddGroup(&cobra.Group{
		ID:    "run",
		Title: "Router Daemon",
	})
	dv.CmdDv.Use = "run config-file"
	dv.CmdDv.Short = "Start the NDN Distance Vector Daemon"
	cmdDv.AddCommand(dv.CmdDv)

	cmdDv.AddGroup(&cobra.Group{
		ID:    "dvc",
		Title: "Router Control",
	})
	for _, sub := range dvc.Cmds() {
		sub.GroupID = "dvc"
		cmdDv.AddCommand(sub)
	}

	return cmdDv
}
