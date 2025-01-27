package face

import enc "github.com/named-data/ndnd/std/encoding"

type Face interface {
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

	// Open starts the face.
	Open() error
	// Close stops the face.
	Close() error
	// Send sends a packet frame to the face.
	Send(pkt enc.Wire) error
}
