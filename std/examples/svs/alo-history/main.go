package main

import (
	"bufio"
	"fmt"
	"os"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/engine"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn/svs_ps"
	"github.com/named-data/ndnd/std/object"
	ndn_sync "github.com/named-data/ndnd/std/sync"
)

func main() {
	// This example shows how to use the SVS ALO with the SnapshotNodeHistory.
	//
	// The SnapshotNodeHistory strategy will deliver all publications since the
	// node bootstrapped. This strategy is useful when the application cannot
	// take a snapshot of its state, and the publication history is important.
	//
	// Before running this example, make sure the strategy is correctly setup
	// to multicast for the /ndn/svs prefix. For example, using the following:
	//
	//   ndnd fw strategy-set prefix=/ndn/svs strategy=/localhost/nfd/strategy/multicast
	//
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <name>", os.Args[0])
		os.Exit(1)
	}

	// Parse command line arguments
	group, _ := enc.NameFromStr("/ndn/svs")
	name, err := enc.NameFromStr(os.Args[1])
	if err != nil {
		log.Fatal(nil, "Invalid node ID", "name", os.Args[1])
		return
	}

	// Create a new engine
	app := engine.NewBasicEngine(engine.NewDefaultFace())
	err = app.Start()
	if err != nil {
		log.Fatal(nil, "Unable to start engine", "err", err)
		return
	}
	defer app.Stop()

	// Create object client
	client := object.NewClient(app, object.NewMemoryStore(), nil)
	if err = client.Start(); err != nil {
		log.Error(nil, "Unable to start object client", "err", err)
		return
	}
	defer client.Stop()

	// Create a new SVS ALO instance
	svsalo := ndn_sync.NewSvsALO(ndn_sync.SvsAloOpts{
		// Name is the name of the node
		Name: name,

		// Svs is the Sync group options
		Svs: ndn_sync.SvSyncOpts{
			Client:      client,
			GroupPrefix: group,
		},

		// This strategy internally takes regular snapshots of the entire history
		// of publications for this node. At the application layer, all publications
		// since the node bootstrapped will be delivered.
		Snapshot: &ndn_sync.SnapshotNodeHistory{
			Client:    client,
			Threshold: 10,
		},
	})

	// OnError gets called when we get an error from the SVS ALO instance.
	svsalo.SetOnError(func(err error) {
		fmt.Fprintf(os.Stderr, "*** %v\n", err)
	})

	// Subscribe to all publications
	svsalo.SubscribePublisher(enc.Name{}, func(pub ndn_sync.SvsPub) {
		// Normal publication, just print it
		if !pub.IsSnapshot {
			fmt.Printf("%s: %s\n", pub.Publisher, pub.Bytes())
			return
		}

		// Snapshot publication, unwrap and print all messages
		snapshot, err := svs_ps.ParseHistorySnap(enc.NewWireView(pub.Content), true)
		if err != nil {
			log.Error(nil, "Unable to parse snapshot", "err", err)
			return
		}
		fmt.Fprintf(os.Stderr, "*** Snapshot from %s with %d entries\n",
			pub.Publisher, len(snapshot.Entries))

		// Print all messages in the snapshot
		for _, entry := range snapshot.Entries {
			fmt.Printf("%s: %s\n", pub.Publisher, entry.Content.Join())
		}
	})

	// Register routes to the local forwarder
	for _, route := range []enc.Name{svsalo.SyncPrefix(), svsalo.DataPrefix()} {
		err = app.RegisterRoute(route)
		if err != nil {
			log.Error(nil, "Unable to register route", "err", err)
			return
		}
		defer app.UnregisterRoute(route)
	}

	if err = svsalo.Start(); err != nil {
		log.Error(nil, "Unable to start SVS ALO", "err", err)
		return
	}
	defer svsalo.Stop()

	fmt.Fprintln(os.Stderr, "*** Joined SVS ALO chat group")
	fmt.Fprintln(os.Stderr, "*** You are:", name)
	fmt.Fprintln(os.Stderr, "*** Type a message and press enter to send.")
	fmt.Fprintln(os.Stderr, "*** Press Ctrl+C to exit.")
	fmt.Fprintln(os.Stderr)

	// Publish an initial message to announce our presence
	_, err = svsalo.Publish(enc.Wire{[]byte("Joined chat")})
	if err != nil {
		log.Error(nil, "Unable to publish message", "err", err)
	}

	counter := 1
	reader := bufio.NewReader(os.Stdin)
	for {
		// Read chat message from stdin
		line, err := reader.ReadBytes('\n')
		if err != nil {
			log.Error(nil, "Unable to read line", "err", err)
			return
		}

		// Trim newline character
		line = line[:len(line)-1]
		if len(line) == 0 {
			continue
		}

		// Special testing function !! to send 20 messages after counter
		if string(line) == "!!" {
			for i := 0; i < 20; i++ {
				_, err = svsalo.Publish(enc.Wire{[]byte(fmt.Sprintf("Message %d", counter))})
				if err != nil {
					log.Error(nil, "Unable to publish message", "err", err)
				}
				counter++
			}
			continue
		}

		// Publish chat message
		_, err = svsalo.Publish(enc.Wire{line})
		if err != nil {
			log.Error(nil, "Unable to publish message", "err", err)
		}
	}
}
