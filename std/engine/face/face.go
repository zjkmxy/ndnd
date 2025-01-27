package face

import enc "github.com/named-data/ndnd/std/encoding"

type Face interface {
	// String returns the log identifier.
	String() string
	// IsRunning returns true if the face is running.
	IsRunning() bool
	// IsLocal returns true if the face is local.
	IsLocal() bool
	// OnPacket sets the callback for receiving packets.
	// This function should only be called by engine implementations.
	OnPacket(onPkt func(frame []byte) error)
	// OnError sets the callback for errors.
	// This function should only be called by engine implementations.
	OnError(onError func(err error) error)

	// Open starts the face and may blocks until it is up.
	Open() error
	// Close stops the face.
	Close() error
	// Send sends a packet frame to the face.
	Send(pkt enc.Wire) error

	// OnUp sets the callback for the face going up.
	// The callback may be called multiple times.
	OnUp(onUp func())
	// OnDown sets the callback for the face going down.
	// The callback may be called multiple times.
	// The callback will not be called when the face is closed.
	OnDown(onDown func())
}
