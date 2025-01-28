package ndncert

import (
	"errors"
)

type ChallengeEmail struct {
	// Email address to send the challenge to.
	Email string
	// Callback to get the code from the user.
	CodeCallback func(status string) string
}

func (*ChallengeEmail) Name() string {
	return "email"
}

func (c *ChallengeEmail) Request(input map[string][]byte, status *string) (map[string][]byte, error) {
	// Validate challenge configuration
	if c.Email == "" || c.CodeCallback == nil {
		return nil, errors.New("email challenge not configured")
	}

	// Initial request parameters
	if input == nil {
		return map[string][]byte{
			"email": []byte(c.Email),
		}, nil
	}

	// Challenge response code
	if status != nil && (*status == "need-code" || *status == "wrong-code") {
		code := c.CodeCallback(*status)
		if code == "" {
			return nil, errors.New("no code provided")
		}

		return map[string][]byte{
			"code": []byte(code),
		}, nil
	}

	// Unknown status
	return nil, errors.New("unknown input to email challenge")
}
