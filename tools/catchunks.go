package tools

import (
	"flag"
	"fmt"
	"os"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/engine"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/object"
)

type CatChunks struct {
	args []string
}

func RunCatChunks(args []string) {
	(&CatChunks{args: args}).run()
}

func (cc *CatChunks) String() string {
	return "cat"
}

func (cc *CatChunks) run() {
	flagset := flag.NewFlagSet("cat", flag.ExitOnError)
	flagset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <name>\n", cc.args[0])
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Retrieves an object with the specified name.\n")
		fmt.Fprintf(os.Stderr, "The object contents are written to stdout on success.\n")
		flagset.PrintDefaults()
	}
	flagset.Parse(cc.args[1:])

	argName := flagset.Arg(0)
	if argName == "" {
		flagset.Usage()
		os.Exit(2)
	}

	name, err := enc.NameFromStr(argName)
	if err != nil {
		log.Fatal(cc, "Invalid name", "name", argName)
		return
	}

	// start face and engine
	app := engine.NewBasicEngine(engine.NewDefaultFace())
	err = app.Start()
	if err != nil {
		log.Fatal(cc, "Unable to start engine", "err", err)
		return
	}
	defer app.Stop()

	// start object client
	cli := object.NewClient(app, object.NewMemoryStore(), nil)
	err = cli.Start()
	if err != nil {
		log.Fatal(cc, "Unable to start object client", "err", err)
		return
	}
	defer cli.Stop()

	done := make(chan ndn.ConsumeState)
	t1, t2 := time.Now(), time.Now()
	byteCount := 0

	// calling Content() on a status object clears the buffer
	// and returns the new data the next time it is called
	write := func(status ndn.ConsumeState) {
		for _, chunk := range status.Content() {
			os.Stdout.Write(chunk)
			byteCount += len(chunk)
		}
	}

	// fetch object
	cli.Consume(name, func(status ndn.ConsumeState) bool {
		if status.IsComplete() {
			t2 = time.Now()
			write(status)
			done <- status
		}

		if status.Progress()%1000 == 0 {
			log.Debug(cc, "Consume progress", "progress", float64(status.Progress())/float64(status.ProgressMax())*100)
			write(status)
		}

		return true
	})
	state := <-done

	if state.Error() != nil {
		log.Fatal(cc, "Error fetching object", "err", state.Error())
		return
	}

	// statistics
	fmt.Fprintf(os.Stderr, "Object fetched %s\n", state.Name())
	fmt.Fprintf(os.Stderr, "Content: %d bytes\n", byteCount)
	fmt.Fprintf(os.Stderr, "Time taken: %s\n", t2.Sub(t1))
	fmt.Fprintf(os.Stderr, "Throughput: %f Mbit/s\n", float64(byteCount*8)/t2.Sub(t1).Seconds()/1e6)
}
