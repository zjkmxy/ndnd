package object

import (
	"fmt"

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
}

// Create a new client with given engine and store
func NewClient(engine ndn.Engine, store ndn.Store, trust *sec.TrustConfig) ndn.Client {
	client := new(Client)
	client.engine = engine
	client.store = store
	client.trust = trust
	client.fetcher = newRrSegFetcher(client)
	return client
}

// Instance log identifier
func (c *Client) String() string {
	return "client"
}

// Start the client. The engine must be running.
func (c *Client) Start() error {
	if !c.engine.IsRunning() {
		return fmt.Errorf("engine is not running")
	}

	if err := c.engine.AttachHandler(enc.Name{}, c.onInterest); err != nil {
		return err
	}

	return nil
}

// Stop the client
func (c *Client) Stop() error {
	if err := c.engine.DetachHandler(enc.Name{}); err != nil {
		return err
	}

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

// IsCongested returns true if the client is congested
func (c *Client) IsCongested() bool {
	return c.fetcher.IsCongested()
}
