package tools

import (
	"fmt"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/utils"
)

func (n *Nfdc) fetchStatusDataset(suffix enc.Name) (enc.Wire, error) {
	n.Start()
	defer n.Stop()

	// TODO: segmented fetch once supported by fw/mgmt
	name := append(n.GetPrefix(), suffix...)
	config := &ndn.InterestConfig{
		MustBeFresh: true,
		CanBePrefix: true,
		Lifetime:    utils.IdPtr(time.Second),
		Nonce:       utils.ConvertNonce(n.engine.Timer().Nonce()),
	}
	interest, err := n.engine.Spec().MakeInterest(name, config, nil, nil)
	if err != nil {
		return nil, err
	}

	ch := make(chan ndn.ExpressCallbackArgs)
	err = n.engine.Express(interest, func(args ndn.ExpressCallbackArgs) {
		ch <- args
		close(ch)
	})
	if err != nil {
		return nil, err
	}

	res := <-ch
	if res.Result != ndn.InterestResultData {
		return nil, fmt.Errorf("interest failed: %d", res.Result)
	}

	return res.Data.Content(), nil
}
