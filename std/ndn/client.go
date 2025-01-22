package ndn

import (
	enc "github.com/named-data/ndnd/std/encoding"
)

// ExpressRArgs are the arguments for the express retry API
type ExpressRArgs struct {
	Name     enc.Name
	Config   *InterestConfig
	AppParam enc.Wire
	Signer   Signer
	Retries  int
	Callback ExpressCallbackFunc
}

type Client interface {
	// Express a single interest with reliability
	ExpressR(args ExpressRArgs, callback ExpressCallbackFunc)
}
