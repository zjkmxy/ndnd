package nfdc

import (
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/object"
)

func (t *Tool) fetchStatusDataset(suffix enc.Name) ([]byte, error) {
	// consume-only client, no need for a store
	client := object.NewClient(t.engine, nil, nil)
	client.Start()
	defer client.Stop()

	ch := make(chan ndn.ConsumeState)
	client.ConsumeExt(ndn.ConsumeExtArgs{
		Name:       t.Prefix().Append(suffix...),
		NoMetadata: true, // NFD has no RDR metadata
		Callback: func(status ndn.ConsumeState) bool {
			if !status.IsComplete() {
				return true
			}
			ch <- status
			close(ch)
			return true
		},
	})

	res := <-ch
	if err := res.Error(); err != nil {
		return nil, err
	}

	return res.Content(), nil
}
