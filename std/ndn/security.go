package ndn

import (
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
)

// Signature is the abstract of the signature of a packet.
// Some of the fields are invalid for Data or Interest.
type Signature interface {
	SigType() SigType
	KeyName() enc.Name
	SigNonce() []byte
	SigTime() *time.Time
	SigSeqNum() *uint64
	Validity() (notBefore, notAfter *time.Time)
	SigValue() []byte
}

// Signer is the interface of a NDN packet signer.
type Signer interface {
	CryptoSigner
	// KeyLocator returns the key name of the signer.
	KeyLocator() enc.Name
}

// SigChecker is a basic function to check the signature of a packet.
// In NTSchema, policies&sub-trees are supposed to be used for validation;
// SigChecker is only designed for low-level engine.
// Create a go routine for time consuming jobs.
type SigChecker func(name enc.Name, sigCovered enc.Wire, sig Signature) bool

// CryptoSigner is the low level interface of a generic signer.
type CryptoSigner interface {
	// SigInfo returns the configuration of the signature.
	Type() SigType
	// EstimateSize gives the approximate size of the signature in bytes.
	EstimateSize() uint
	// Sign computes the signature value of a wire.
	Sign(enc.Wire) ([]byte, error)
}
