package ndn

import (
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
)

type Client interface {
	// Instance log identifier
	String() string
	// Start the client. The engine must be running.
	Start() error
	// Stop the client.
	Stop() error
	// Underlying API engine
	Engine() Engine
	// Undelying data store
	Store() Store
	// Produce and sign data, and insert into the client's store.
	// The input data will be freed as the object is segmented.
	Produce(args ProduceArgs) (enc.Name, error)
	// Remove an object from the client's store by name
	Remove(name enc.Name) error
	// Consume an object with a given name
	Consume(name enc.Name, callback ConsumeCallback)
	// ConsumeExt is a more advanced consume API that allows for
	// more control over the fetching process.
	ConsumeExt(args ConsumeExtArgs)
	// Express a single interest with reliability
	ExpressR(args ExpressRArgs)
}

// ProduceArgs are the arguments for the produce API
type ProduceArgs struct {
	// name of the object to produce
	Name enc.Name
	// raw data contents
	Content enc.Wire
	// version of the object (defaults to unix timestamp, 0 for immutable)
	Version *uint64
	// time for which the object version can be cached (default 4s)
	FreshnessPeriod time.Duration
	// do not create metadata packet
	NoMetadata bool
}

// callback for consume API
// return true to continue fetching the object
type ConsumeCallback func(status ConsumeState) bool

// ConsumeState is the state of the consume operation
type ConsumeState interface {
	// Name of the object being consumed
	Name() enc.Name
	// Error that occurred during fetching
	Error() error
	// IsComplete returns true if the content has been completely fetched
	IsComplete() bool
	// Content is the currently available buffer in the content
	// any subsequent calls to Content() will return data after the previous call
	Content() []byte
	// Progress counter
	Progress() int
	// ProgressMax is the max value for the progress counter (-1 for unknown)
	ProgressMax() int
}

// ConsumeExtArgs are arguments for the ConsumeExt API
type ConsumeExtArgs struct {
	// name of the object to consume
	Name enc.Name
	// callback when data is available
	Callback ConsumeCallback
	// do not fetch metadata packet (advanced usage)
	NoMetadata bool
}

// ExpressRArgs are the arguments for the express retry API
type ExpressRArgs struct {
	Name     enc.Name
	Config   *InterestConfig
	AppParam enc.Wire
	Signer   Signer
	Retries  int
	Callback ExpressCallbackFunc
}
