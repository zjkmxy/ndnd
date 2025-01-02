package tools

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
	"github.com/named-data/ndnd/std/utils"
)

func (n *Nfdc) ExecCmd(mod string, cmd string, args []string) {
	ctrlArgs := mgmt.ControlArgs{}

	for _, arg := range args[1:] {
		kv := strings.SplitN(arg, "=", 2)
		if len(kv) != 2 {
			fmt.Fprintf(os.Stderr, "Invalid argument: %s (should be key=value)\n", arg)
			return
		}
		n.convCmdArg(&ctrlArgs, kv[0], kv[1])
	}

	raw, err := n.engine.ExecMgmtCmd(mod, cmd, &ctrlArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %+v\n", err)
		return
	}

	res, ok := raw.(*mgmt.ControlResponse)
	if !ok {
		fmt.Fprintf(os.Stderr, "Invalid response type: %T\n", raw)
		return
	}

	if res.Val == nil || res.Val.Params == nil {
		fmt.Fprintf(os.Stderr, "Empty response: %+v\n", res)
		return
	}

	fmt.Printf("Status=%d (%s)\n", res.Val.StatusCode, res.Val.StatusText)

	params := res.Val.Params.ToDict()
	for key, val := range params {
		fmt.Printf("  %s=%v\n", key, val)
	}
}

func (n *Nfdc) convCmdArg(ctrlArgs *mgmt.ControlArgs, key string, val string) {
	switch key {
	case "face":
		if v, err := strconv.ParseUint(val, 10, 64); err == nil {
			ctrlArgs.FaceId = utils.IdPtr(v)
		} else {
			fmt.Fprintf(os.Stderr, "Invalid face ID: %s\n", val) // TODO: support URI
			os.Exit(1)
		}
	case "remote":
		ctrlArgs.Uri = utils.IdPtr(val)
	case "local":
		ctrlArgs.LocalUri = utils.IdPtr(val)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command argument key: %s\n", key)
		os.Exit(1)
	}
}
