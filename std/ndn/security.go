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
	// SigInfo returns the configuration of the signature.
	Type() SigType
	// KeyName returns the key name of the signer.
	KeyName() enc.Name
	// EstimateSize gives the approximate size of the signature in bytes.
	EstimateSize() uint
	// Sign computes the signature value of a wire.
	Sign(enc.Wire) ([]byte, error)
	// Public returns the public key of the signer or nil.
	Public() ([]byte, error)
}

// SigChecker is a basic function to check the signature of a packet.
// In NTSchema, policies&sub-trees are supposed to be used for validation;
// SigChecker is only designed for low-level engine.
// Create a go routine for time consuming jobs.
type SigChecker func(name enc.Name, sigCovered enc.Wire, sig Signature) bool

// KeyChain is the interface of a keychain.
type KeyChain interface {
	// GetIdentity returns the identity by full name.
	GetIdentity(enc.Name) Identity
	// InsertKey inserts a key to the keychain.
	InsertKey(Signer) error
	// InsertCert inserts a certificate to the keychain.
	InsertCert([]byte) error
}

// Identity is the interface of a signing identity.
type Identity interface {
	// Name returns the full name of the identity.
	Name() enc.Name
	// Signer returns the default signer of the identity.
	Signer() Signer
	// AllSigners returns all signers of the identity.
	AllSigners() []Signer
}
