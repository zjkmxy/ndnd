package tools

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/engine"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	sec "github.com/named-data/ndnd/std/security"
	"github.com/named-data/ndnd/std/security/ndncert"
	spec_ndncert "github.com/named-data/ndnd/std/security/ndncert/tlv"
	sig "github.com/named-data/ndnd/std/security/signer"
)

type CertClient struct {
	args []string
	opts struct {
		keyFile   string
		outFile   string
		challenge string
		email     string
		noprobe   bool
	}

	caCert    []byte
	signer    ndn.Signer
	challenge ndncert.Challenge
}

func RunCertClient(args []string) {
	(&CertClient{args: args}).run()
}

func (c *CertClient) String() string {
	return "ndncert-cli"
}

func (c *CertClient) run() {
	flagset := flag.NewFlagSet("ndncert-client", flag.ExitOnError)
	flagset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <ca-file>\n", c.args[0])
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Interactive client for the NDNCERT CA.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "   ca-file string\n")
		fmt.Fprintf(os.Stderr, "        File with CA root certificate\n")
		flagset.PrintDefaults()
	}

	flagset.StringVar(&c.opts.outFile, "o", "", "Output filename without extension (default: stdout)")
	flagset.StringVar(&c.opts.keyFile, "k", "", "File with NDN key to certify (default: generate new key)")
	flagset.StringVar(&c.opts.challenge, "c", "", "Challenge type (default: ask)")
	flagset.StringVar(&c.opts.email, "email", "", "Email address for probe and email challenge")
	flagset.BoolVar(&c.opts.noprobe, "no-probe", false, "Skip probe and use the provided key directly")
	flagset.Parse(c.args[1:])

	argCaCert := flagset.Arg(0)
	if flagset.NArg() != 1 || argCaCert == "" {
		flagset.Usage()
		os.Exit(3)
	}

	// Read CA certificate
	caCertFile, err := os.ReadFile(argCaCert)
	if err != nil {
		log.Fatal(c, "Unable to read CA certificate file", "file", argCaCert)
		return
	}
	_, caCerts, _ := sec.DecodeFile(caCertFile)
	if len(caCerts) != 1 {
		log.Fatal(c, "CA certificate file must contain exactly one certificate", "file", argCaCert)
		return
	}
	c.caCert = caCerts[0]

	// Read private key if specified
	if keyFile := c.opts.keyFile; keyFile != "" {
		keyBytes, err := os.ReadFile(keyFile)
		if err != nil {
			log.Fatal(c, "Unable to read private key file", "file", keyFile)
			return
		}
		keys, _, _ := sec.DecodeFile(keyBytes)
		if len(keys) != 1 {
			log.Fatal(c, "Private key file must contain exactly one key", "file", keyFile)
			return
		}
		c.signer = keys[0]
	}

	// Choose challenge type
	c.challenge = c.chooseChallenge()

	// Run the client
	c.client()
}

// chooseChallenge asks the user to choose a challenge type.
// It then prompts the user for the necessary information and returns the challenge.
func (c *CertClient) chooseChallenge() ndncert.Challenge {
	defer fmt.Fprintln(os.Stderr)

	challenges := []string{ndncert.KwEmail, ndncert.KwPin}

	if c.opts.challenge == "" {
		i := c.chooseOpts("Please choose a challenge type:", challenges)
		c.opts.challenge = challenges[i]
	}

	switch c.opts.challenge {
	case ndncert.KwEmail:
		if c.opts.email == "" {
			c.scanln("Enter your email address", &c.opts.email)
		}
		return &ndncert.ChallengeEmail{
			Email: c.opts.email,
			CodeCallback: func(status string) (code string) {
				fmt.Fprintf(os.Stderr, "\n")
				fmt.Fprintf(os.Stderr, "Challenge Status: %s\n", status)
				fmt.Fprintf(os.Stderr, "Enter the code sent to your email address: ")
				fmt.Scanln(&code)
				return code
			},
		}

	case ndncert.KwPin:
		return &ndncert.ChallengePin{
			CodeCallback: func(status string) (code string) {
				fmt.Fprintf(os.Stderr, "\n")
				fmt.Fprintf(os.Stderr, "Challenge Status: %s\n", status)
				fmt.Fprintf(os.Stderr, "Enter the secret PIN: ")
				fmt.Scanln(&code)
				return code
			},
		}

	default:
		fmt.Fprintf(os.Stderr, "Invalid challenge selected: %s\n", c.opts.challenge)
		os.Exit(3)
	}

	return nil
}

func (c *CertClient) client() {
	// Start the engine
	engine := engine.NewBasicEngine(engine.NewDefaultFace())
	err := engine.Start()
	if err != nil {
		log.Fatal(c, "Unable to start engine", "err", err)
		return
	}
	defer engine.Stop()

	// Create ndncert client
	certClient, err := ndncert.NewClient(engine, c.caCert)
	if err != nil {
		log.Fatal(c, "Unable to create ndncert client", "err", err)
		return
	}

	// Set signer if provided
	if c.signer != nil {
		certClient.SetSigner(c.signer)
	}

	// Start the certificate request
	certRes, err := certClient.RequestCert(ndncert.RequestCertArgs{
		Challenge: c.challenge,
		OnProfile: func(profile *spec_ndncert.CaProfile) error {
			c.printCaProfile(profile)
			return nil
		},
		DisableProbe: c.opts.noprobe,
		OnProbeParam: func(key string) ([]byte, error) {
			switch key {
			case ndncert.KwEmail:
				if c.opts.email == "" {
					c.scanln("Enter your email address", &c.opts.email)
				}
				return []byte(c.opts.email), nil

			default:
				var val string
				c.scanln(fmt.Sprintf("Enter probing parameter '%s'", key), &val)
				return []byte(val), nil
			}
		},
		OnChooseKey: func(suggestions []enc.Name) int {
			suggestionsStr := make([]string, 0, len(suggestions))
			for _, sgst := range suggestions {
				suggestionsStr = append(suggestionsStr, sgst.String())
			}
			return c.chooseOpts("Please choose a key name:", suggestionsStr)
		},
		OnKeyChosen: func(keyName enc.Name) error {
			fmt.Fprintf(os.Stderr, "Certifying key: %s\n", keyName)
			return nil
		},
	})
	if err != nil {
		// Handle mismatched key name
		var pmErr ndncert.ErrSignerProbeMismatch
		if errors.As(err, &pmErr) {
			fmt.Fprintf(os.Stderr, "Key name does not match CA probe response:\n")
			fmt.Fprintf(os.Stderr, "  %s\n", pmErr.KeyName)
			fmt.Fprintf(os.Stderr, "CA suggestions:\n")
			for _, sgst := range pmErr.Suggested {
				fmt.Fprintf(os.Stderr, "  %s\n", sgst)
			}
			os.Exit(1)
			return
		}

		// Handle no key suggestions from PROBE step
		if errors.Is(err, ndncert.ErrNoKeySuggestions) {
			fmt.Fprintf(os.Stderr, "No key suggestions from the CA\n")
			fmt.Fprintf(os.Stderr, "Please provide a key file with -k\n")
			os.Exit(1)
			return
		}

		// Handle other errors
		log.Fatal(c, err.Error())
		return
	}

	// PEM encode the certificate
	certBytes, err := sec.PemEncode(certRes.CertWire.Join())
	if err != nil {
		log.Fatal(c, "Unable to PEM encode certificate", "err", err)
		return
	}

	// Marshal the key if not specified as file
	var keyBytes []byte = nil
	if c.opts.keyFile == "" {
		keyWire, err := sig.MarshalSecret(certRes.Signer)
		if err != nil {
			log.Fatal(c, "Unable to marshal key", "err", err)
			return
		}

		keyBytes, err = sec.PemEncode(keyWire.Join())
		if err != nil {
			log.Error(c, "Unable to PEM encode key", "err", err)
		}
	}

	if c.opts.outFile == "" {
		// Write the key and certificate to stdout
		if len(keyBytes) > 0 {
			os.Stdout.Write(keyBytes)
			os.Stdout.Write([]byte("\n"))
		}
		os.Stdout.Write(certBytes)
	} else {
		// Write the key to a file
		if len(keyBytes) > 0 {
			err := os.WriteFile(c.opts.outFile+".key", keyBytes, 0600)
			if err != nil {
				log.Fatal(c, "Unable to write key file", "err", err)
				return
			}
		}

		// Write the certificate to a file
		err := os.WriteFile(c.opts.outFile+".cert", certBytes, 0644)
		if err != nil {
			log.Fatal(c, "Unable to write certificate file", "err", err)
			return
		}
	}
}

func (c *CertClient) chooseOpts(msg string, opts []string) int {
	fmt.Fprintf(os.Stderr, "%s\n", msg)
	for i, opt := range opts {
		fmt.Fprintf(os.Stderr, "  %d. %s\n", i+1, opt)
	}
	fmt.Fprintf(os.Stderr, "Choice: ")

	var choice string
	fmt.Scanln(&choice)

	// search by name
	for i, opt := range opts {
		if choice == opt {
			return i
		}
	}

	// search by index
	index, err := strconv.Atoi(choice)
	if err == nil {
		if index >= 1 && index <= len(opts) {
			return index - 1
		}
	}

	fmt.Fprintf(os.Stderr, "Invalid choice: %s\n\n", choice)

	return c.chooseOpts(msg, opts)
}

func (c *CertClient) printCaProfile(profile *spec_ndncert.CaProfile) {
	fmt.Fprintln(os.Stderr, "=============== CA Profile ================")
	fmt.Fprintln(os.Stderr, profile.CaInfo)
	fmt.Fprintln(os.Stderr, "Name:", profile.CaPrefix.Name)
	fmt.Fprintln(os.Stderr, "Max Validity:", time.Duration(profile.MaxValidPeriod)*time.Second)
	fmt.Fprintln(os.Stderr, "Probe Keys:", profile.ParamKey)
	fmt.Fprintln(os.Stderr, "===========================================")
	fmt.Fprintln(os.Stderr)
}

func (c *CertClient) scanln(msg string, val *string) {
	fmt.Fprintf(os.Stderr, "%s: ", msg)
	fmt.Scanln(val)
	if *val == "" {
		fmt.Fprintf(os.Stderr, "Invalid value entered\n")
		os.Exit(3)
	}
}
