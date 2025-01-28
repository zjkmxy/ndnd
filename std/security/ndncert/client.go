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
	engine   ndn.Engine
	caPrefix enc.Name
	signer   ndn.Signer

	client  ndn.Client
	ecdhKey *ecdh.PrivateKey
	aeadCtr *AeadCounter

	// caPrefix            string
	// caPublicIdentityKey *ecdsa.PublicKey
	// certKey             *ecdsa.PrivateKey
	// certRequestBytes    []byte
	// challengeStatus     ChallengeStatus
	// ecdhState           *ECDHState
	// interestSigner      ndn.Signer
	// ndnEngine           ndn.Engine

	// requestId                   RequestId
	// counterInitializationVector *CounterInitializationVector
	// serverBlockCounter          *uint32
	// symmetricKey                [16]byte
}

func NewClient(engine ndn.Engine, caPrefix enc.Name, signer ndn.Signer) (*Client, error) {
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

	return &Client{
		engine:   engine,
		caPrefix: caPrefix,
		signer:   signer,

		client:  client,
		ecdhKey: ecdhKey,
		aeadCtr: NewAeadCounter(),
	}, nil
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

	ch := make(chan ndn.ExpressCallbackArgs)
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
		return nil, fmt.Errorf("failed to fetch probe response: %s", args.Result)
	}

	content := args.Data.Content()
	if err := IsError(content); err != nil {
		return nil, err
	}

	return tlv.ParseProbeRes(enc.NewWireReader(content), false)
}

func (c *Client) New(challenge Challenge) (*tlv.NewRes, error) {
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

	ch := make(chan ndn.ExpressCallbackArgs)
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
		return nil, fmt.Errorf("failed NEW fetch: %s", args.Result)
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
	var prevParams map[string][]byte = nil
	var prevStatus *string = nil

	if prevRes != nil {
		prevStatus = prevRes.ChalStatus

		switch ChallengeStatus(prevRes.Status) {
		case ChallengeStatusChallenge:
			// Always provide params (even if they are empty) to the challenge
			prevParams = prevRes.Params
			if prevParams == nil {
				prevParams = make(map[string][]byte)
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
	ch := make(chan ndn.ExpressCallbackArgs)
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
		return nil, fmt.Errorf("failed CHALLENGE fetch: %s", args.Result)
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
