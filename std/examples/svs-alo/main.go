package main

import (
	"bufio"
	"fmt"
	"os"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/engine"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/object"
	ndn_sync "github.com/named-data/ndnd/std/sync"
)

func main() {
	// Before running this example, make sure the strategy is correctly setup
	// to multicast for the /ndn/svs prefix. For example, using the following:
	//
	//   ndnd fw strategy set prefix=/ndn/svs strategy=/localhost/nfd/strategy/multicast
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

	// Register routes to the local forwarder
	for _, route := range []enc.Name{group, name} {
		err = app.RegisterRoute(route)
		if err != nil {
			log.Error(nil, "Unable to register route", "err", err)
			return
		}
		defer app.UnregisterRoute(route)
	}

	// Total number and size of messages
	msgCount := 0
	msgSize := 0

	// Start SVS instance
	svsalo := ndn_sync.NewSvsALO(ndn_sync.SvsAloOpts{
		// Name is the name of the node
		Name: name,

		// Svs is the Sync group options
		Svs: ndn_sync.SvSyncOpts{
			Client:      client,
			GroupPrefix: group,
		},

		// The snapshot strategy provides a way to prevent slowly aggregating
		// old publications to arrive at the application's current state.
		//
		// The SnapshotNodeLatest strategy will only deliver the latest snapshot
		// and all publications after the snapshot. As a result, the snapshot
		// in this case should contain the entire state of the node.
		Snapshot: &ndn_sync.SnapshotNodeLatest{
			Client: client,
			SnapMe: func() (enc.Wire, error) {
				// In this example, we will ignore the old messages
				// and only send a message with the total number of messages.
				//
				// A real application can bundle all state of the node in
				// the snapshot publication.
				message := fmt.Sprintf("[Snapshot] Skipping %d messages with %d bytes", msgCount, msgSize)

				return enc.Wire{[]byte(message)}, nil
			},
			Threshold: 10,
		},
	})
	if err = svsalo.Start(); err != nil {
		log.Error(nil, "Unable to start SVS ALO", "err", err)
		return
	}
	defer svsalo.Stop()

	// Subscribe to all messages
	svsalo.SubscribePublisher(enc.Name{}, func(pub ndn_sync.SvsPub) {
		// Both normal and snapshot publications will be received here.
		// Snapshots will have the IsSnapshot flag set to true.
		// The publications will be received in the order they were published.
		//
		// Since the snapshot strategy is set to SnapshotNodeLatest.,
		// older publications before a snapshot will be ignored.
		fmt.Printf("%s: %s\n", pub.Publisher, pub.Bytes())
	})

	fmt.Fprintln(os.Stderr, "Joined SVS ALO chat group")
	fmt.Fprintln(os.Stderr, "You are:", name)
	fmt.Fprintln(os.Stderr, "Type a message and press enter to send.")
	fmt.Fprintln(os.Stderr, "Press Ctrl+C to exit.")
	fmt.Fprintln(os.Stderr)

	// Publish initial message
	msgCount++
	_, err = svsalo.Publish(enc.Wire{[]byte("Joined the chatroom")})
	if err != nil {
		log.Error(nil, "Unable to publish message", "err", err)
	}

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
		msgCount++
		msgSize += len(line)

		// Publish chat message
		_, err = svsalo.Publish(enc.Wire{line})
		if err != nil {
			log.Error(nil, "Unable to publish message", "err", err)
		}
	}
}
