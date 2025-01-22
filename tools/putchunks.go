package tools

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/engine"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/object"
)

type PutChunks struct {
	args []string
}

func RunPutChunks(args []string) {
	(&PutChunks{args: args}).run()
}

func (pc *PutChunks) String() string {
	return "put"
}

func (pc *PutChunks) run() {
	flagset := flag.NewFlagSet("put", flag.ExitOnError)
	flagset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <name>\n", pc.args[0])
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Publish data under the specified prefix.\n")
		fmt.Fprintf(os.Stderr, "This tool expects data from the standard input.\n")
		flagset.PrintDefaults()
	}
	flagset.Parse(pc.args[1:])

	argName := flagset.Arg(0)
	if argName == "" {
		flagset.Usage()
		os.Exit(2)
	}

	name, err := enc.NameFromStr(argName)
	if err != nil {
		log.Fatal(pc, "Invalid object name", "name", argName)
		return
	}

	// start face and engine
	app := engine.NewBasicEngine(engine.NewDefaultFace())
	err = app.Start()
	if err != nil {
		log.Fatal(pc, "Unable to start engine", "err", err)
		return
	}
	defer app.Stop()

	// start object client
	cli := object.NewClient(app, object.NewMemoryStore(), nil)
	err = cli.Start()
	if err != nil {
		log.Fatal(pc, "Unable to start object client", "err", err)
		return
	}
	defer cli.Stop()

	// read from stdin till eof
	var content enc.Wire
	for {
		buf := make([]byte, 8192)
		n, err := io.ReadFull(os.Stdin, buf)
		if n > 0 {
			content = append(content, buf[:n])
		}
		if err != nil {
			break
		}
	}

	// produce object
	vname, err := cli.Produce(ndn.ProduceArgs{
		Name:    name,
		Content: content,
	})
	if err != nil {
		log.Fatal(pc, "Unable to produce object", "err", err)
		return
	}

	content = nil // gc
	log.Info(pc, "Object produced", "name", vname)

	// register route to the object
	err = app.RegisterRoute(name)
	if err != nil {
		log.Fatal(pc, "Unable to register route", "err", err)
		return
	}

	// wait forever
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)
	receivedSig := <-sigchan
	log.Info(nil, "Received signal - exiting", "signal", receivedSig)
}
