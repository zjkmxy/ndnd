// This example uses the old schema implemementation and does not work
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/apex/log"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/schema"
	"github.com/zjkmxy/go-ndn/pkg/schema/demo"
	sec "github.com/zjkmxy/go-ndn/pkg/security"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

var app *basic_engine.Engine
var tree *schema.Tree

const HmacKey = "Hello, World!"

func passAll(enc.Name, enc.Wire, ndn.Signature) bool {
	return true
}

func main() {
	log.SetLevel(log.InfoLevel)
	logger := log.WithField("module", "main")

	// Setup schema tree
	tree = &schema.Tree{}
	path, _ := enc.NamePatternFromStr("/randomData/<v=time>")
	node := &schema.LeafNode{}
	err := tree.PutNode(path, node)
	if err != nil {
		logger.Fatalf("Unable to construst the schema tree: %+v", err)
		return
	}
	node.Set(schema.PropCanBePrefix, false)
	node.Set(schema.PropMustBeFresh, true)
	node.Set(schema.PropLifetime, 6*time.Second)
	node.Set(schema.PropFreshness, 1*time.Second)
	node.Set(schema.PropValidDuration, 876000*time.Hour)
	schema.AddEventListener(node, schema.PropOnGetDataSigner, func(enc.Matching, enc.Name, schema.Context) ndn.Signer {
		return sec.NewSha256Signer()
	})
	passAllChecker := func(enc.Matching, enc.Name, ndn.Signature, enc.Wire, schema.Context) schema.ValidRes {
		return schema.VrPass
	}
	node.Get(schema.PropOnValidateData).(*schema.Event[*schema.NodeValidateEvent]).Add(&passAllChecker)
	path, _ = enc.NamePatternFromStr("/contentKey")
	ckNode := &demo.ContentKeyNode{}
	err = tree.PutNode(path, ckNode)
	if err != nil {
		logger.Fatalf("Unable to construst the schema tree: %+v", err)
		return
	}
	demo.NewFixedKeySigner([]byte(HmacKey)).Apply(ckNode)

	// Setup policies
	memStorage := demo.NewMemStoragePolicy()
	memStorage.Apply(node)
	memStorage.Apply(ckNode)
	demo.NewRegisterPolicy().Apply(tree.Root)

	// Start engine
	timer := basic_engine.NewTimer()
	face := basic_engine.NewStreamFace("unix", "/var/run/nfd.sock", true)
	app = basic_engine.NewEngine(face, timer, sec.NewSha256IntSigner(timer), passAll)
	err = app.Start()
	if err != nil {
		logger.Fatalf("Unable to start engine: %+v", err)
		return
	}
	defer app.Shutdown()

	// Attach schema
	prefix, _ := enc.NameFromStr("/example/schema/encryptionApp")
	err = tree.Attach(prefix, app)
	if err != nil {
		logger.Fatalf("Unable to attach the schema to the engine: %+v", err)
		return
	}
	defer tree.Detach()

	// Produce data
	ver := utils.MakeTimestamp(timer.Now())
	ckid := ckNode.GenKey(enc.Matching{})
	cipherText, err := ckNode.Encrypt(enc.Matching{}, ckid, enc.Wire{[]byte("Hello, world!")})
	if err != nil {
		logger.Fatalf("Unable to encrypt data: %+v", err)
		return
	}
	node.Provide(enc.Matching{
		"time": ver,
	}, nil, cipherText, schema.Context{})
	fmt.Printf("Generated packet with version= %d\n", ver)

	// Wait for keyboard quit signal
	sigChannel := make(chan os.Signal, 1)
	fmt.Print("Start serving ...\n")
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)
	receivedSig := <-sigChannel
	logger.Infof("Received signal %+v - exiting\n", receivedSig)
}
