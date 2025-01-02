package tools

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	enc "github.com/named-data/ndnd/std/encoding"
	mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
	"github.com/named-data/ndnd/std/utils"
)

func (n *Nfdc) ExecCmd(mod string, cmd string, args []string, defaults []string) {
	// parse command arguments
	ctrlArgs := mgmt.ControlArgs{}

	// set default values first, then user-provided values
	for _, arg := range append(defaults, args[1:]...) {
		kv := strings.SplitN(arg, "=", 2)
		if len(kv) != 2 {
			fmt.Fprintf(os.Stderr, "Invalid argument: %s (should be key=value)\n", arg)
			return
		}
		n.convCmdArg(&ctrlArgs, kv[0], kv[1])
	}

	// execute command
	raw, execErr := n.engine.ExecMgmtCmd(mod, cmd, &ctrlArgs)
	if raw == nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %+v\n", execErr)
		return
	}

	// parse response
	res, ok := raw.(*mgmt.ControlResponse)
	if !ok {
		fmt.Fprintf(os.Stderr, "Invalid response type: %T\n", raw)
		return
	}

	if res.Val == nil || res.Val.Params == nil {
		fmt.Fprintf(os.Stderr, "Empty response: %+v\n", res)
		return
	}

	// print status code and text
	fmt.Printf("Status=%d (%s)\n", res.Val.StatusCode, res.Val.StatusText)

	// iterate over parameters in sorted order
	params := res.Val.Params.ToDict()
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// print parameters
	for _, key := range keys {
		val := params[key]

		// convert some values to human-readable form
		switch key {
		case "FacePersistency":
			val = mgmt.Persistency(val.(uint64)).String()
		}

		fmt.Printf("  %s=%v\n", key, val)
	}

	if execErr != nil {
		os.Exit(1)
	}
}

func (n *Nfdc) convCmdArg(ctrlArgs *mgmt.ControlArgs, key string, val string) {
	// helper function to parse uint64 values
	parseUint := func(val string) uint64 {
		v, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid value for %s: %s\n", key, val)
			os.Exit(9)
		}
		return v
	}

	// convert key-value pairs to command arguments
	switch key {
	// face arguments
	case "face":
		ctrlArgs.FaceId = utils.IdPtr(parseUint(val))
	case "remote":
		ctrlArgs.Uri = utils.IdPtr(val)
	case "local":
		ctrlArgs.LocalUri = utils.IdPtr(val)
	case "mtu":
		ctrlArgs.Mtu = utils.IdPtr(parseUint(val))
	case "persistency":
		switch val {
		case "permanent":
			ctrlArgs.FacePersistency = utils.IdPtr(uint64(mgmt.PersistencyPermanent))
		case "persistent":
			ctrlArgs.FacePersistency = utils.IdPtr(uint64(mgmt.PersistencyPersistent))
		default:
			fmt.Fprintf(os.Stderr, "Invalid persistency: %s\n", val)
			os.Exit(9)
		}

	// route arguments
	case "name":
		name, err := enc.NameFromStr(val)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid name for %s: %s\n", key, val)
			os.Exit(9)
		}
		ctrlArgs.Name = name
	case "cost":
		ctrlArgs.Cost = utils.IdPtr(parseUint(val))
	case "origin":
		ctrlArgs.Origin = utils.IdPtr(parseUint(val))
	case "expires":
		ctrlArgs.ExpirationPeriod = utils.IdPtr(parseUint(val))

	// unknown argument
	default:
		fmt.Fprintf(os.Stderr, "Unknown command argument key: %s\n", key)
		os.Exit(9)
	}
}
