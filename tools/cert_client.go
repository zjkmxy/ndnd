package tools

import (
	"crypto/elliptic"
	"flag"
	"fmt"
	"os"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/engine"
	"github.com/named-data/ndnd/std/log"
	sec "github.com/named-data/ndnd/std/security"
	"github.com/named-data/ndnd/std/security/ndncert"
	"github.com/named-data/ndnd/std/security/signer"
)

type CertClient struct {
	args []string
	ca   enc.Name
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
		fmt.Fprintf(os.Stderr, "Usage: %s <ca-prefix>\n", c.args[0])
		flagset.PrintDefaults()
	}
	flagset.Parse(c.args[1:])

	argCa := flagset.Arg(0)
	if argCa == "" {
		flagset.Usage()
		os.Exit(3)
	}

	ca, err := enc.NameFromStr(argCa)
	if err != nil {
		log.Fatal(c, "Invalid prefix", "name", argCa)
		return
	}
	c.ca = ca

	// Start the engine
	eng := engine.NewBasicEngine(engine.NewDefaultFace())
	err = eng.Start()
	if err != nil {
		log.Fatal(c, "Unable to start engine", "err", err)
		return
	}
	defer eng.Stop()

	// TODO: use provided key from CLI
	name, _ := enc.NameFromStr("/ndn/edu/ucla/varunpatil")
	keyName := sec.MakeKeyName(name)
	signer, _ := signer.KeygenEcc(keyName, elliptic.P256())

	// Create ndncert client
	ncli, err := ndncert.NewClient(eng, c.ca, signer)
	if err != nil {
		log.Fatal(c, "Unable to create ndncert client", "err", err)
		return
	}

	profile, err := ncli.FetchProfile()
	if err != nil {
		panic(err)
	}
	fmt.Println(profile.CaInfo)
	fmt.Println(profile.CaPrefix.Name)
	fmt.Println(profile.MaxValidPeriod)
	fmt.Println(profile.ParamKey)

	challenge := &ndncert.ChallengeEmail{
		Email: "varunpatil@ucla.edu",
		CodeCallback: func(status string) string {
			fmt.Printf("challenge status: %s\n", status)
			var code string
			fmt.Print("Enter the code: ")
			fmt.Scanln(&code)
			return code
		},
	}

	probe, err := ncli.FetchProbe(challenge)
	if err != nil {
		panic(err)
	}
	fmt.Println(probe.RedirectPrefix)
	for _, val := range probe.Vals {
		fmt.Println(val.Response)
		fmt.Println(*val.MaxSuffixLength)
	}

	newRes, err := ncli.New(challenge)
	if err != nil {
		panic(err)
	}

	chRes, err := ncli.Challenge(challenge, newRes, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(chRes.CertName.Name)
}
