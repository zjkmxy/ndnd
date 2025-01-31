package ndn

import (
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
)

// ObjectVersionImmutable is the version number for immutable objects.
// A version number of 0 will be used on the wire.
const VersionImmutable = uint64(1<<63 - 15)

// ObjectVersionUnixMicro is the version number for objects with a unix timestamp.
// A version number of microseconds since the unix epoch will be used on the wire.
// Current unix time must be positive to use this.
const VersionUnixMicro = uint64(1<<63 - 16)

// Client is the interface for the Object Client API
type Client interface {
	// String is the instance log identifier.
	String() string
	// Start starts the client. The engine must be running.
	Start() error
	// Stop stops the client.
	Stop() error
	// Engine gives the underlying API engine.
	Engine() Engine
	// Store gives the underlying data store.
	Store() Store
	// Produce generates and signs data, and inserts into the client's store.
	// The input data will be freed as the object is segmented.
	// Returns the final versioned name of the object.
	Produce(args ProduceArgs) (enc.Name, error)
	// Remove removes an object from the client's store by name.
	Remove(name enc.Name) error
	// Consume fetches an object with a given name
	Consume(name enc.Name, callback func(status ConsumeState))
	// ConsumeExt is a more advanced consume API that allows for
	// more control over the fetching process.
	ConsumeExt(args ConsumeExtArgs)
	// ExpressR sends a single interest with reliability.
	// Since this is a low-level API, the result is NOT validated.
	ExpressR(args ExpressRArgs)
	// Suggest suggests a signer for a given name.
	// nil is returned if no signer is found.
	SuggestSigner(name enc.Name) Signer
	// Validate validates a single data packet.
	Validate(data Data, sigCov enc.Wire, callback func(bool, error))
	// ValidateExt validates a single data packet (advanced API).
	ValidateExt(args ValidateExtArgs)
}

// ProduceArgs are the arguments for the produce API.
type ProduceArgs struct {
	// Name is the name of the object to produce.
	Name enc.Name
	// Content is the raw data wire.
	// Content can be larger than a single packet and will be segmented.
	Content enc.Wire
	// Version of the object. This option is required.
	// Available magic values are:
	//   VersionImmutable for v=0
	//   VersionUnixMicro for v=current unix time in microseconds
	Version uint64
	// Time for which the object version can be cached (default 4s).
	FreshnessPeriod time.Duration
	// NoMetadata disables RDR metadata (advanced usage).
	NoMetadata bool
}

// ConsumeState is the state of the consume operation
type ConsumeState interface {
	// Name of the object being consumed.
	Name() enc.Name
	// Error that occurred during fetching.
	Error() error
	// IsComplete returns true if the content has been completely fetched.
	IsComplete() bool
	// Content is the currently available buffer in the content.
	// any subsequent calls to Content() will return data after the previous call.
	Content() enc.Wire
	// Progress counter
	Progress() int
	// ProgressMax is the max value for the progress counter (-1 for unknown).
	ProgressMax() int
	// Cancel the consume operation.
	Cancel()
}

// ConsumeExtArgs are arguments for the ConsumeExt API.
type ConsumeExtArgs struct {
	// Name is the name of the object to consume.
	Name enc.Name
	// Callback is called when data is available.
	// True should be returned to continue fetching the object.
	Callback func(status ConsumeState)
	// NoMetadata disables fetching RDR metadata (advanced usage).
	NoMetadata bool
}

// ExpressRArgs are the arguments for the express retry API.
type ExpressRArgs struct {
	// Name of the data to fetch.
	Name enc.Name
	// Interest configuration.
	Config *InterestConfig
	// AppParam for the interest.
	AppParam enc.Wire
	// Signer for signed interests.
	Signer Signer
	// Number of retries.
	Retries int
	// Callback for the result. This will be called on the engine's
	// main thread, so make sure it is either non-blocking and very fast,
	// or use a goroutine to handle the result.
	Callback ExpressCallbackFunc
}

// ValidateExtArgs are the arguments for the advanced validate API.
type ValidateExtArgs struct {
	// Data packet to validate.
	Data Data
	// Signature covered wire.
	SigCovered enc.Wire
	// Callback for the result.
	Callback func(bool, error)
	// Override data name during first validation.
	OverrideName enc.Name
	// Next Hop ID to use for fetching certificates.
	CertNextHop *uint64
}
