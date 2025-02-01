package ndncert

import "fmt"

type ChallengePin struct {
	// Callback to get the code from the user.
	CodeCallback func(status string) string
}

func (*ChallengePin) Name() string {
	return KwPin
}

func (c *ChallengePin) Request(input ParamMap, status *string) (ParamMap, error) {
	// Validate challenge configuration
	if c.CodeCallback == nil {
		return nil, fmt.Errorf("pin challenge not configured")
	}

	// Initial request parameters
	if input == nil {
		return ParamMap{}, nil
	}

	// Challenge response code
	if status != nil && (*status == "need-code" || *status == "wrong-code") {
		code := c.CodeCallback(*status)
		if code == "" {
			return nil, fmt.Errorf("no code provided")
		}

		return ParamMap{
			KwCode: []byte(code),
		}, nil
	}

	// Unknown status
	return nil, fmt.Errorf("unknown input to pin challenge")
}
