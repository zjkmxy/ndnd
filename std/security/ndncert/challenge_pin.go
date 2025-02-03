package ndncert

import (
	"errors"

	enc "github.com/named-data/ndnd/std/encoding"
)

type ChallengePin struct {
	// Callback to get the code from the user.
	CodeCallback func(status string) string
}

func (*ChallengePin) Name() string {
	return KwPin
}

func (c *ChallengePin) Request(input ParamMap, status enc.Optional[string]) (ParamMap, error) {
	// Validate challenge configuration
	if c.CodeCallback == nil {
		return nil, errors.New("pin challenge not configured")
	}

	// Initial request parameters
	if input == nil {
		return ParamMap{}, nil
	}

	// Challenge response code
	if s := status.GetOr(""); s == "need-code" || s == "wrong-code" {
		code := c.CodeCallback(s)
		if code == "" {
			return nil, errors.New("no code provided")
		}

		return ParamMap{
			KwCode: []byte(code),
		}, nil
	}

	// Unknown status
	return nil, errors.New("unknown input to pin challenge")
}
