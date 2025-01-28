package tools

import (
	"crypto/elliptic"
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
		keyFile, err := os.ReadFile(keyFile)
		if err != nil {
			log.Fatal(c, "Unable to read private key file", "file", keyFile)
			return
		}
		keys, _, _ := sec.DecodeFile(keyFile)
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

	// Fetch root CA profile
	caprefix := certClient.CaPrefix()
	profile, err := certClient.FetchProfile()
	if err != nil {
		log.Fatal(c, "Unable to fetch CA profile", "err", err)
		return
	}
	c.printCaProfile(profile)

	// Probe is optional, if disabled use the provided key directly
	probe := &spec_ndncert.ProbeRes{}

	// Probe the CA and get key suggestions
	if !c.opts.noprobe {
		// We expect all CAs to support the same param keys for now.
		// This is a reasonable assumption (for now) at least on testbed.
		probeParams := ndncert.ParamMap{}
		for _, paramKey := range profile.ParamKey {
			switch paramKey {
			case ndncert.KwEmail:
				if c.opts.email == "" {
					c.scanln("Enter email address for PROBE", &c.opts.email)
				}
				probeParams[paramKey] = []byte(c.opts.email)

			default:
				var paramVal string
				c.scanln(fmt.Sprintf("Enter PROBE param '%s'", paramKey), &paramVal)
				probeParams[paramKey] = []byte(paramVal)
			}
		}

		// Probe the CA and redirect to the correct CA
		probe, err = certClient.FetchProbeRedirect(probeParams)
		if err != nil {
			log.Fatal(c, "Unable to probe the CA", "err", err)
			return
		}

		// Fetch redirected CA profile if changed
		if !certClient.CaPrefix().Equal(caprefix) {
			fmt.Fprintf(os.Stderr, "Redirected to CA: %s\n\n", certClient.CaPrefix())
			profile, err = certClient.FetchProfile()
			if err != nil {
				log.Fatal(c, "Unable to fetch CA profile", "err", err)
				return
			}
			c.printCaProfile(profile)
		}
	}

	// If a key is provided, check if the name matches
	if c.signer != nil {
		// if no suggestions, assume it's correct
		found := len(probe.Vals) == 0

		// find the key name in the suggestions
		keyName := c.signer.KeyName()
		for _, sgst := range probe.Vals {
			if sgst.Response.IsPrefix(keyName) {
				found = true
				break
			}
		}

		// if not found, print suggestions and exit
		if !found {
			fmt.Fprintf(os.Stderr, "Key name does not match CA probe response:\n")
			fmt.Fprintf(os.Stderr, "  %s\n", keyName)
			fmt.Fprintf(os.Stderr, "CA suggestions:\n")
			for _, sgst := range probe.Vals {
				fmt.Fprintf(os.Stderr, "  %s\n", sgst.Response)
			}
			os.Exit(1)
		}
	} else {
		// If no key is provided, generate one from the suggestions
		var identity enc.Name

		if len(probe.Vals) == 0 {
			// If no suggestions, print error and exit
			fmt.Fprintf(os.Stderr, "No key suggestions from the CA\n")
			fmt.Fprintf(os.Stderr, "Please provide a key file with -k\n")
			os.Exit(1)
		} else if len(probe.Vals) == 1 {
			// If only one suggestion, use it
			identity = probe.Vals[0].Response
		} else {
			// If multiple suggestions, ask the user to choose
			idNames := make([]string, 0, len(probe.Vals))
			for _, sgst := range probe.Vals {
				idNames = append(idNames, sgst.Response.String())
			}
			idx := c.chooseOpts("Please choose a key name:", idNames)
			identity = probe.Vals[idx].Response
		}

		// Generate key
		keyName := sec.MakeKeyName(identity)
		c.signer, err = sig.KeygenEcc(keyName, elliptic.P256())
		if err != nil {
			log.Fatal(c, "Unable to generate key", "err", err)
		}
	}

	// Print name of key we are finalizing
	certClient.SetSigner(c.signer)
	fmt.Fprintf(os.Stderr, "Certifying key: %s\n", c.signer.KeyName())

	// Start a new certification request
	// Use the longest possible validity period
	newRes, err := certClient.New(c.challenge, time.Now().Add(time.Second*time.Duration(profile.MaxValidPeriod)))
	if err != nil {
		log.Fatal(c, "Unable to start new certification request", "err", err)
		return
	}

	// Complete the challenge
	chRes, err := certClient.Challenge(c.challenge, newRes, nil)
	if err != nil {
		log.Fatal(c, "Unable to complete challenge", "err", err)
		return
	}
	if chRes.CertName.Name == nil {
		log.Fatal(c, "No certificate issued", "err", err)
		return
	}

	fmt.Fprintf(os.Stderr, "Issued certificate: %s\n", chRes.CertName.Name)
	fmt.Fprintln(os.Stderr)

	// Get the certificate
	_, certWire, err := certClient.FetchIssuedCert(chRes)
	if err != nil {
		log.Fatal(c, "Unable to fetch certificate", "err", err)
		return
	}
	certBytes, err := sec.PemEncode(certWire.Join())
	if err != nil {
		log.Fatal(c, "Unable to PEM encode certificate", "err", err)
		return
	}

	// Marshal the key if not specified as file
	var keyBytes []byte = nil
	if c.opts.keyFile == "" {
		keyWire, err := sig.MarshalSecret(c.signer)
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
