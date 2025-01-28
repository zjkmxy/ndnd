package ndncert

import (
	"crypto/ecdh"
	"fmt"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/object"
	sec "github.com/named-data/ndnd/std/security"
	"github.com/named-data/ndnd/std/security/ndncert/tlv"
	sig "github.com/named-data/ndnd/std/security/signer"
)

const RequestIdLength = 8

type ChallengeResult struct {
	ChallengeStatus       *ChallengeStatus
	RemainingTime         *uint64
	RemainingTries        *uint64
	IssuedCertificateName enc.Name
	ForwardingHint        enc.Name
	IssuedCertificateBits *[]byte
	ErrorMessage          *tlv.ErrorRes
}

type Client struct {
	engine ndn.Engine
	signer ndn.Signer

	caCert   ndn.Data
	caPrefix enc.Name

	client  ndn.Client
	ecdhKey *ecdh.PrivateKey
	aeadCtr *AeadCounter
}

// NewClient creates a new NDNCERT client.
//
//	engine: NDN engine
//	caCert: CA certificate raw wire
//	signer: signer for the client
func NewClient(engine ndn.Engine, caCert []byte) (*Client, error) {
	// Decode CA certificate
	cert, _, err := engine.Spec().ReadData(enc.NewBufferReader(caCert))
	if err != nil {
		return nil, fmt.Errorf("failed to decode CA certificate: %w", err)
	}

	// Object API client for network requests. We use our own client here
	// because this needs a custom trust model that is not the same as app.
	// No need to start the client because it is only used for consume.
	// TODO: find a better way to express this and prevent the error
	// returned by start due to multiple clients.
	// TODO: validate packets received by the client using the key.
	client := object.NewClient(engine, object.NewMemoryStore(), nil)

	// Generate ECDH Key used for encryption
	ecdhKey, err := EcdhKeygen()
	if err != nil {
		return nil, err
	}

	// Get CA prefix from CA certificate
	caPrefix, err := sec.GetIdentityFromCertName(cert.Name())
	if err != nil {
		return nil, fmt.Errorf("invalid CA certificate name: %w", err)
	}

	return &Client{
		engine: engine,

		caCert:   cert,
		caPrefix: caPrefix,

		client:  client,
		ecdhKey: ecdhKey,
		aeadCtr: NewAeadCounter(),
	}, nil
}

// SetSigner sets the signer for the client.
func (c *Client) SetSigner(signer ndn.Signer) {
	c.signer = signer
}

// FetchProfile fetches the profile from the CA (blocking).
func (c *Client) FetchProfile() (*tlv.CaProfile, error) {
	ch := make(chan ndn.ConsumeState)
	c.client.Consume(c.caPrefix.Append(
		enc.NewStringComponent(enc.TypeGenericNameComponent, "CA"),
		enc.NewStringComponent(enc.TypeGenericNameComponent, "INFO"),
	), func(status ndn.ConsumeState) {
		if status.IsComplete() {
			ch <- status
		}
	})
	state := <-ch
	if err := state.Error(); err != nil {
		return nil, err
	}

	return tlv.ParseCaProfile(enc.NewWireReader(state.Content()), false)
}

// FetchProbe sends a PROBE request to the CA (blocking).
func (c *Client) FetchProbe(challenge Challenge) (*tlv.ProbeRes, error) {
	params, err := challenge.Request(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get challenge params: %w", err)
	}

	probeParams := tlv.ProbeReq{Params: params}

	ch := make(chan ndn.ExpressCallbackArgs, 1)
	c.client.ExpressR(ndn.ExpressRArgs{
		Name: c.caPrefix.Append(
			enc.NewStringComponent(enc.TypeGenericNameComponent, "CA"),
			enc.NewStringComponent(enc.TypeGenericNameComponent, "PROBE"),
		),
		Config: &ndn.InterestConfig{
			CanBePrefix: false,
			MustBeFresh: true,
		},
		AppParam: probeParams.Encode(),
		Signer:   sig.NewSha256Signer(),
		Retries:  3,
		Callback: func(args ndn.ExpressCallbackArgs) { ch <- args },
	})
	args := <-ch
	if args.Result != ndn.InterestResultData {
		return nil, fmt.Errorf("failed to fetch probe response: %s (%+v)", args.Result, args.Error)
	}

	content := args.Data.Content()
	if err := IsError(content); err != nil {
		return nil, err
	}

	return tlv.ParseProbeRes(enc.NewWireReader(content), false)
}

// FetchProbeRedirect sends a PROBE request to the CA (blocking).
// If a redirect is received, the request is sent to the new location.
func (c *Client) FetchProbeRedirect(challenge Challenge) (probe *tlv.ProbeRes, err error) {
	for i := 0; i < 4; i++ {
		probe, err = c.FetchProbe(challenge)
		if err != nil {
			return nil, err
		}

		// Check if the probe response is a redirect
		if probe.RedirectPrefix == nil ||
			len(probe.RedirectPrefix.Name) == 0 ||
			probe.RedirectPrefix.Name.Equal(c.caPrefix) {
			// Found last CA in the chain
			return probe, nil
		}

		// Redirect to a different CA
		caCert := probe.RedirectPrefix.Name
		caPrefix, err := sec.GetIdentityFromCertName(caCert)
		if err != nil {
			return nil, fmt.Errorf("invalid redirect %s: %w", caCert, err)
		}

		// Update the client to use the new CA
		// TODO: fetch the CA certificate and update caCert
		c.caPrefix = caPrefix
	}

	return nil, fmt.Errorf("too many redirects")
}

// New sends a NEW request to the CA (blocking).
func (c *Client) New(challenge Challenge) (*tlv.NewRes, error) {
	// Signer must be set before this step
	if c.signer == nil {
		return nil, fmt.Errorf("signer not set")
	}

	// Generate self-signed cert as CSR
	// TODO: validity period is a parameter
	csr, err := sec.SelfSign(sec.SignCertArgs{
		Signer:    c.signer,
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(0, 0, 3),
	})
	if err != nil {
		return nil, err
	}

	// Send NEW request to CA
	newParams := tlv.NewReq{
		EcdhPub: c.ecdhKey.Public().(*ecdh.PublicKey).Bytes(),
		CertReq: csr,
	}

	ch := make(chan ndn.ExpressCallbackArgs, 1)
	c.client.ExpressR(ndn.ExpressRArgs{
		Name: c.caPrefix.Append(
			enc.NewStringComponent(enc.TypeGenericNameComponent, "CA"),
			enc.NewStringComponent(enc.TypeGenericNameComponent, "NEW"),
		),
		Config: &ndn.InterestConfig{
			CanBePrefix: false,
			MustBeFresh: true,
		},
		AppParam: newParams.Encode(),
		Signer:   c.signer,
		Retries:  3,
		Callback: func(args ndn.ExpressCallbackArgs) { ch <- args },
	})
	args := <-ch
	if args.Result != ndn.InterestResultData {
		return nil, fmt.Errorf("failed NEW fetch: %s (%+v)", args.Result, args.Error)
	}

	content := args.Data.Content()
	if err := IsError(content); err != nil {
		return nil, fmt.Errorf("failed NEW: %+v", err)
	}

	newRes, err := tlv.ParseNewRes(enc.NewWireReader(content), false)
	if err != nil {
		return nil, err
	}

	// Check if challenge is supported
	hasChallenge := false
	for _, chName := range newRes.Challenge {
		if chName == challenge.Name() {
			hasChallenge = true
			break
		}
	}
	if !hasChallenge {
		return nil, fmt.Errorf("challenge not supported by CA: %s", challenge.Name())
	}

	return newRes, nil
}

// Challenge sends a CHALLENGE request to the CA (blocking).
func (c *Client) Challenge(
	challenge Challenge,
	newRes *tlv.NewRes,
	prevRes *tlv.ChallengeRes,
) (*tlv.ChallengeRes, error) {
	var prevParams ParamMap = nil
	var prevStatus *string = nil

	if prevRes != nil {
		prevStatus = prevRes.ChalStatus

		switch ChallengeStatus(prevRes.Status) {
		case ChallengeStatusChallenge:
			// Always provide params (even if they are empty) to the challenge
			prevParams = prevRes.Params
			if prevParams == nil {
				prevParams = make(ParamMap)
			}
		default:
			return nil, fmt.Errorf("invalid challenge status: %d", prevRes.Status)
		}
	}

	// Get challenge params
	params, err := challenge.Request(prevParams, prevStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to get challenge params: %w", err)
	}

	// Create CHALLENGE request for CA
	chParams := tlv.ChallengeReq{
		Challenge: challenge.Name(),
		Params:    params,
	}

	// Derive symmetric key using ECDH-HKDF
	symkey, err := EcdhHkdf(c.ecdhKey, newRes.EcdhPub, newRes.Salt, newRes.ReqId)
	if err != nil {
		return nil, fmt.Errorf("failed to derive symmetric key: %w", err)
	}

	// Encrypt and send CHALLENGE request
	chParamsC, err := AeadEncrypt(
		[16]byte(symkey), chParams.Encode().Join(),
		newRes.ReqId, c.aeadCtr)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt CHALLENGE: %w", err)
	}

	// ExpressR will resign on failure, we don't want this to happen
	// TODO: add an option to ExpressR to not resign
	ch := make(chan ndn.ExpressCallbackArgs, 1)
	c.client.ExpressR(ndn.ExpressRArgs{
		Name: c.caPrefix.Append(
			enc.NewStringComponent(enc.TypeGenericNameComponent, "CA"),
			enc.NewStringComponent(enc.TypeGenericNameComponent, "CHALLENGE"),
			enc.NewBytesComponent(enc.TypeGenericNameComponent, newRes.ReqId),
		),
		Config: &ndn.InterestConfig{
			CanBePrefix: false,
			MustBeFresh: true,
		},
		AppParam: chParamsC.TLV().Encode(),
		Signer:   c.signer,
		Retries:  3,
		Callback: func(args ndn.ExpressCallbackArgs) { ch <- args },
	})
	args := <-ch
	if args.Result != ndn.InterestResultData {
		return nil, fmt.Errorf("failed CHALLENGE fetch: %s (%+v)", args.Result, args.Error)
	}

	content := args.Data.Content()
	if err := IsError(content); err != nil {
		return nil, fmt.Errorf("failed CHALLENGE: %+v", err)
	}

	chResEnc, err := tlv.ParseCipherMsg(enc.NewWireReader(content), false)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CHALLENGE cipher response: %w", err)
	}

	chResAeadMsg := AeadMessage{}
	chResAeadMsg.FromTLV(chResEnc)
	chResBytes, err := AeadDecrypt([16]byte(symkey), chResAeadMsg, newRes.ReqId)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt CHALLENGE response: %w", err)
	}

	chRes, err := tlv.ParseChallengeRes(enc.NewBufferReader(chResBytes), false)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CHALLENGE response: %w", err)
	}

	switch ChallengeStatus(chRes.Status) {
	case ChallengeStatusBefore:
		return chRes, ErrChallengeBefore
	case ChallengeStatusChallenge:
		// Continue with the challenge
		return c.Challenge(challenge, newRes, chRes)
	case ChallengeStatusPending:
		// TODO: likely need to wait and retry
		return chRes, ErrChallengePending
	case ChallengeStatusSuccess:
		return chRes, nil
	case ChallengeStatusFailure:
		return chRes, ErrChallengeFailed
	default:
		return chRes, ErrChallengeStatusUnknown
	}
}

func (c *Client) FetchCert(chRes *tlv.ChallengeRes) (ndn.Data, enc.Wire, error) {
	if chRes.Status != uint64(ChallengeStatusSuccess) {
		return nil, nil, fmt.Errorf("invalid challenge status: %d", chRes.Status)
	}

	if chRes.CertName == nil {
		return nil, nil, fmt.Errorf("missing certificate name")
	}

	// Challenge response may contain a forwarding hint
	var fwHint []enc.Name
	if chRes.ForwardingHint != nil {
		fwHint = []enc.Name{chRes.ForwardingHint.Name}
	}

	// Fetch issued certificate
	ch := make(chan ndn.ExpressCallbackArgs, 1)
	c.client.ExpressR(ndn.ExpressRArgs{
		Name: chRes.CertName.Name,
		Config: &ndn.InterestConfig{
			CanBePrefix:    false,
			ForwardingHint: fwHint,
		},
		Retries:  3,
		Callback: func(args ndn.ExpressCallbackArgs) { ch <- args },
	})
	args := <-ch
	if args.Result != ndn.InterestResultData {
		return nil, nil, fmt.Errorf("failed to fetch certificate: %s (%+v)", args.Result, args.Error)
	}

	return args.Data, args.RawData, nil
}
