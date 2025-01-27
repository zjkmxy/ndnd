package object

import (
	"errors"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	sec "github.com/named-data/ndnd/std/security"
)

type Client struct {
	// underlying API engine
	engine ndn.Engine
	// data storage
	store ndn.Store
	// trust configuration
	trust *sec.TrustConfig
	// segment fetcher
	fetcher rrSegFetcher

	// stop the client
	stop chan bool
	// outgoing interest pipeline
	outpipe chan ndn.ExpressRArgs
	// [fetcher] incoming data pipeline
	seginpipe chan rrSegHandleDataArgs
	// [fetcher] queue for new object fetch
	segfetch chan *ConsumeState
	// [validate] queue for new object validation
	validatepipe chan sec.TrustConfigValidateArgs
}

// Create a new client with given engine and store
func NewClient(engine ndn.Engine, store ndn.Store, trust *sec.TrustConfig) ndn.Client {
	client := new(Client)
	client.engine = engine
	client.store = store
	client.trust = trust
	client.fetcher = newRrSegFetcher(client)

	client.stop = make(chan bool)
	client.outpipe = make(chan ndn.ExpressRArgs, 512)
	client.seginpipe = make(chan rrSegHandleDataArgs, 512)
	client.segfetch = make(chan *ConsumeState, 128)
	client.validatepipe = make(chan sec.TrustConfigValidateArgs, 64)

	return client
}

// Instance log identifier
func (c *Client) String() string {
	return "client"
}

// Start the client. The engine must be running.
func (c *Client) Start() error {
	if !c.engine.IsRunning() {
		return errors.New("client start when engine not running")
	}

	if err := c.engine.AttachHandler(enc.Name{}, c.onInterest); err != nil {
		return err
	}

	go c.run()
	return nil
}

// Stop the client
func (c *Client) Stop() error {
	c.stop <- true
	return nil
}

// Get the underlying engine
func (c *Client) Engine() ndn.Engine {
	return c.engine
}

// Get the underlying store
func (c *Client) Store() ndn.Store {
	return c.store
}

// Get the client interface
func (c *Client) Client() ndn.Client {
	return c
}

// Main goroutine for all client processing
func (c *Client) run() {
	for {
		select {
		case <-c.stop:
			return
		case args := <-c.outpipe:
			c.expressRImpl(args)
		case args := <-c.seginpipe:
			c.fetcher.handleData(args.args, args.state)
		case state := <-c.segfetch:
			c.fetcher.add(state)
		case args := <-c.validatepipe:
			c.trust.Validate(args)
		}
	}
}
