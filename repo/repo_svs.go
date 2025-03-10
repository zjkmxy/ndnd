package repo

import (
	"fmt"
	"time"

	"github.com/named-data/ndnd/repo/tlv"
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	ndn_sync "github.com/named-data/ndnd/std/sync"
)

type RepoSvs struct {
	config *Config
	client ndn.Client
	cmd    *tlv.SyncJoin
	svsalo *ndn_sync.SvsALO
}

func NewRepoSvs(config *Config, client ndn.Client, cmd *tlv.SyncJoin) *RepoSvs {
	return &RepoSvs{
		config: config,
		client: client,
		cmd:    cmd,
		svsalo: nil,
	}
}

func (r *RepoSvs) String() string {
	return fmt.Sprintf("repo-svs (%s)", r.cmd.Group)
}

func (r *RepoSvs) Start() (err error) {
	log.Info(r, "Starting SVS")

	// Parse snapshot configuration
	var snapshot ndn_sync.Snapshot = nil

	// History snapshot
	if r.cmd.HistorySnapshot != nil {
		if t := r.cmd.HistorySnapshot.Threshold; t < 10 {
			return fmt.Errorf("invalid history snapshot threshold: %d", t)
		}

		snapshot = &ndn_sync.SnapshotNodeHistory{
			Client:    r.client,
			Threshold: r.cmd.HistorySnapshot.Threshold,
			IsRepo:    true,
		}
	}

	// Start SVS ALO
	r.svsalo, err = ndn_sync.NewSvsALO(ndn_sync.SvsAloOpts{
		Name:         enc.Name{enc.NewKeywordComponent("repo")}, // unused
		InitialState: r.readState(),
		Svs: ndn_sync.SvSyncOpts{
			Client:            r.client,
			GroupPrefix:       r.cmd.Group.Name,
			SuppressionPeriod: 500 * time.Millisecond,
			PeriodicTimeout:   365 * 24 * time.Hour, // basically never
			Passive:           true,
		},
		Snapshot: snapshot,
	})
	if err != nil {
		return err
	}

	// Set error handler
	r.svsalo.SetOnError(func(err error) {
		log.Error(r, "SVS ALO error", "err", err)
	})

	// Subscribe to all publishers
	r.svsalo.SubscribePublisher(enc.Name{}, func(pub ndn_sync.SvsPub) {
		r.commitState(pub.State)
	})

	// This covers both the sync prefix and all producers' data prefixes.
	r.client.AnnouncePrefix(ndn.Announcement{
		Name:    r.svsalo.GroupPrefix(),
		Cost:    1000,
		Expose:  true,
		OnError: nil, // TODO
	})

	// Start SVS ALO
	if err = r.svsalo.Start(); err != nil {
		return err
	}

	return nil
}

func (r *RepoSvs) Stop() (err error) {
	log.Info(r, "Stopping SVS")
	if r.svsalo == nil {
		return nil
	}

	// Withdraw group prefix.
	r.client.WithdrawPrefix(r.svsalo.GroupPrefix(), nil)

	// Stop SVS ALO
	if err = r.svsalo.Stop(); err != nil {
		return err
	}

	return nil
}

func (r *RepoSvs) commitState(state enc.Wire) {
	name := r.cmd.Group.Name.Append(enc.NewKeywordComponent("alo-state"))
	r.client.Store().Put(name, state.Join())
}

func (r *RepoSvs) readState() enc.Wire {
	name := r.cmd.Group.Name.Append(enc.NewKeywordComponent("alo-state"))
	if stateWire, _ := r.client.Store().Get(name, false); stateWire != nil {
		return enc.Wire{stateWire}
	}
	return nil
}
