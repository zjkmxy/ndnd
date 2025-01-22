package security

import (
	"fmt"
	"sync"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	spec "github.com/named-data/ndnd/std/ndn/spec_2022"
	"github.com/named-data/ndnd/std/security/signer"
)

// TrustConfig is the configuration of the trust module.
type TrustConfig struct {
	// mutex is the lock for keychain.
	mutex sync.RWMutex
	// keychain is the keychain.
	keychain ndn.KeyChain
	// schema is the trust schema.
	schema ndn.TrustSchema
	// roots are the full names of the trust anchors.
	roots []enc.Name
}

// NewTrustConfig creates a new TrustConfig.
// ALl roots must be full names and already present in the keychain.
func NewTrustConfig(keyChain ndn.KeyChain, schema ndn.TrustSchema, roots []enc.Name) (*TrustConfig, error) {
	// Check arguments
	if keyChain == nil || schema == nil {
		return nil, fmt.Errorf("keychain and schema must not be nil")
	}

	// Check if we have some roots
	if len(roots) == 0 {
		return nil, fmt.Errorf("no trust anchors provided")
	}

	// Check if all roots are present in the keychain
	for _, root := range roots {
		if cert, _ := keyChain.Store().Get(root, false); cert == nil {
			return nil, fmt.Errorf("trust anchor not found in keychain: %s", root)
		}
	}

	return &TrustConfig{
		mutex:    sync.RWMutex{},
		keychain: keyChain,
		schema:   schema,
		roots:    roots,
	}, nil
}

// Suggest suggests a signer for a given name.
func (tc *TrustConfig) Suggest(name enc.Name) ndn.Signer {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()

	return tc.schema.Suggest(name, tc.keychain)
}

// ValidateArgs are the arguments for the Validate function.
type ValidateArgs struct {
	Data       ndn.Data
	DataSigCov enc.Wire

	Fetch func(enc.Name, *ndn.InterestConfig,
		func(data ndn.Data, wire enc.Wire, sigCov enc.Wire, err error))
	Callback func(bool, error)
	DataName enc.Name

	cert       ndn.Data
	certSigCov enc.Wire

	depth int
}

// Validate validates a Data packet using a fetch API.
func (tc *TrustConfig) Validate(args ValidateArgs) {
	if args.Data == nil {
		args.Callback(false, fmt.Errorf("data is nil"))
		return
	}

	if len(args.DataSigCov) == 0 {
		args.Callback(false, fmt.Errorf("data sig covered is nil"))
		return
	}

	// Prevent infinite recursion for signer loops
	if args.depth == 0 {
		args.depth = 32
	} else if args.depth <= 1 {
		args.Callback(false, fmt.Errorf("max depth reached"))
		return
	} else {
		args.depth--
	}

	// Make sure the data is signed
	signature := args.Data.Signature()
	if signature == nil {
		args.Callback(false, fmt.Errorf("signature is nil"))
		return
	}

	// If a certificate is provided, go directly to validation
	if args.cert != nil {
		certName := args.cert.Name()
		dataName := args.Data.Name()
		if len(args.DataName) > 0 {
			dataName = args.DataName
		}

		// Check if the data claims to be a root certificate.
		// This breaks the recursion for validation.
		if dataName.Equal(certName) {
			for _, root := range tc.roots {
				if dataName.Equal(root) {
					args.Callback(true, nil)
					return
				}
			}
			args.Callback(false, fmt.Errorf("data claims to be a trust anchor: %s", dataName))
			return
		}

		// Check schema if the key is allowed
		if !tc.schema.Check(dataName, certName) {
			args.Callback(false, fmt.Errorf("key is not allowed: %s signed by %s", dataName, certName))
			return
		}

		// Validate signature on data
		valid, err := signer.ValidateData(args.Data, args.DataSigCov, args.cert)
		if !valid || err != nil {
			args.Callback(false, fmt.Errorf("signature is invalid: %s (%+v)", args.Data.Name(), err))
			return
		}

		if len(args.certSigCov) == 0 {
			args.Callback(false, fmt.Errorf("cert sig covered is nil: %s", certName))
			return
		}

		// Recursively validate the certificate
		tc.Validate(ValidateArgs{
			Data:       args.cert,
			DataSigCov: args.certSigCov,

			Fetch:    args.Fetch,
			Callback: args.Callback,
			DataName: nil,

			cert:       nil,
			certSigCov: nil,

			depth: args.depth,
		})
		return
	}

	// Get the key locator
	keyLocator := signature.KeyName()
	if keyLocator == nil {
		args.Callback(false, fmt.Errorf("key locator is nil"))
		return
	}

	// Detect if this is a self-signed certificate, and automatically pick the cert
	// as itself to verify in this case.
	if args.Data.ContentType() != nil && *args.Data.ContentType() == ndn.ContentTypeKey && keyLocator.IsPrefix(args.Data.Name()) {
		args.cert = args.Data
		tc.Validate(args)
		return
	}

	// Attempt to get cert from store
	certBytes, err := tc.keychain.Store().Get(keyLocator, true)
	if err != nil {
		log.Error(nil, "Failed to get certificate from store", "error", err)
		args.Callback(false, err)
		return // store is likely broken
	}
	if len(certBytes) > 0 {
		// Attempt to parse the certificate
		args.cert, args.certSigCov, err = spec.Spec{}.ReadData(enc.NewBufferReader(certBytes))
		if err != nil {
			log.Error(nil, "Failed to parse certificate in store", "error", err)
			args.cert = nil
			args.certSigCov = nil
		}
	}

	// Make sure the certificate is fresh
	if args.cert != nil && CertIsExpired(args.cert) {
		args.cert = nil
		args.certSigCov = nil
	}

	// If not found, attempt to fetch cert from network
	if args.cert == nil {
		args.Fetch(keyLocator, &ndn.InterestConfig{
			CanBePrefix: true,
			MustBeFresh: true,
		}, func(cert ndn.Data, wire enc.Wire, sigCov enc.Wire, err error) {
			if err != nil {
				args.Callback(false, err)
				return // failed to fetch cert
			}

			// Bail if the fetched cert is not fresh
			if CertIsExpired(cert) {
				args.Callback(false, fmt.Errorf("certificate is expired: %s", cert.Name()))
				return
			}

			// Call again with the fetched cert
			args.cert = cert
			args.certSigCov = sigCov

			// Monkey patch the callback to store the cert in keychain
			// if the validation passes.
			origCallback := args.Callback
			args.Callback = func(valid bool, err error) {
				if valid && err == nil {
					tc.mutex.Lock()
					err := tc.keychain.InsertCert(wire.Join())
					tc.mutex.Unlock()
					if err != nil {
						log.Error(tc.keychain, "Failed to insert certificate", "error", err)
					}
				}
				origCallback(valid, err) // continue validation
			}

			tc.Validate(args)
		})
	} else {
		tc.Validate(args)
	}
}
