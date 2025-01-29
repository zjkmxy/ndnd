package tools

import "github.com/spf13/cobra"

var toolCat = CatChunks{}
var CmdCatChunks = &cobra.Command{
	GroupID: "tools",
	Use:     "cat name",
	Short:   "Retrieve object under a name prefix",
	Long: `Retrieve an object with the specified name.
The object contents are written to stdout on success.`,
	Args:    cobra.ExactArgs(1),
	Example: `  ndnd cat /my/example/data > data.bin`,
	Run:     toolCat.run,
}

var toolPut = PutChunks{}
var CmdPutChunks = &cobra.Command{
	GroupID: "tools",
	Use:     "put name",
	Short:   "Publish data under a name prefix",
	Long: `Publish data under a name prefix.
This tool expects data from the standard input.`,
	Args:    cobra.ExactArgs(1),
	Example: `  ndnd put /my/example/data < data.bin`,
	Run:     toolPut.run,
}

var toolPingClient = PingClient{}
var CmdPingClient = &cobra.Command{
	GroupID: "tools",
	Use:     "ping name",
	Short:   "Send Interests to a ping server",
	Long: `Ping a name prefix using Interests like /prefix/ping/number
The numbers in the Interests are randomly generated`,
	Args:    cobra.ExactArgs(1),
	Example: `  ndnd ping /my/prefix -c 5`,
	Run:     toolPingClient.run,
}

var toolPingServer = PingServer{}
var CmdPingServer = &cobra.Command{
	GroupID: "tools",
	Use:     "pingserver name",
	Short:   "Start a ping server under a name prefix",
	Args:    cobra.ExactArgs(1),
	Example: `  ndnd pingserver /my/prefix`,
	Run:     toolPingServer.run,
}

func init() {
	CmdPutChunks.Flags().BoolVar(&toolPut.expose, "expose", false, "Use client origin for prefix registration")

	CmdPingClient.Flags().IntVarP(&toolPingClient.interval, "interval", "i", 1000, "ping interval, in milliseconds")
	CmdPingClient.Flags().IntVarP(&toolPingClient.timeout, "timeout", "t", 4000, "timeout for each ping, in milliseconds")
	CmdPingClient.Flags().IntVarP(&toolPingClient.count, "count", "c", 0, "number of pings to send")
	CmdPingClient.Flags().Uint64Var(&toolPingClient.seq, "seq", 0, "start sequence number")
}
