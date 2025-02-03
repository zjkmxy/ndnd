package ndncert

import (
	"errors"

	"github.com/named-data/ndnd/std/types/optional"
)

type ChallengeEmail struct {
	// Email address to send the challenge to.
	Email string
	// Callback to get the code from the user.
	CodeCallback func(status string) string
}

func (*ChallengeEmail) Name() string {
	return KwEmail
}

func (c *ChallengeEmail) Request(input ParamMap, status optional.Optional[string]) (ParamMap, error) {
	// Validate challenge configuration
	if len(c.Email) == 0 || c.CodeCallback == nil {
		return nil, errors.New("email challenge not configured")
	}

	// Initial request parameters
	if input == nil {
		return ParamMap{
			KwEmail: []byte(c.Email),
		}, nil
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
	return nil, errors.New("unknown input to email challenge")
}
