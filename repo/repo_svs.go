package repo

import (
	"fmt"
	"time"

	"github.com/named-data/ndnd/repo/tlv"
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
	ndn_sync "github.com/named-data/ndnd/std/sync"
	"github.com/named-data/ndnd/std/types/optional"
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
			GroupPrefix:       r.cmd.Group,
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

	// Register route to group prefix.
	// This covers both the sync prefix and all producers' data prefixes.
	if err = r.registerRoute(r.svsalo.GroupPrefix()); err != nil {
		return err
	}

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

	// Unregister route to group prefix.
	r.unregisterRoutes(r.svsalo.GroupPrefix())

	// Stop SVS ALO
	if err = r.svsalo.Stop(); err != nil {
		return err
	}

	return nil
}

func (r *RepoSvs) commitState(state enc.Wire) {
	name := r.cmd.Group.Append(enc.NewKeywordComponent("alo-state"))
	r.client.Store().Put(name, state.Join())
}

func (r *RepoSvs) readState() enc.Wire {
	name := r.cmd.Group.Append(enc.NewKeywordComponent("alo-state"))
	if stateWire, _ := r.client.Store().Get(name, false); stateWire != nil {
		return enc.Wire{stateWire}
	}
	return nil
}

func (r *RepoSvs) registerRoute(prefix enc.Name) (err error) {
	if _, err = r.client.Engine().ExecMgmtCmd("rib", "register", &mgmt.ControlArgs{
		Name:   prefix,
		Cost:   optional.Some(uint64(1000)),
		Origin: optional.Some(uint64(mgmt.RouteOriginClient)),
	}); err != nil {
		log.Error(r, "Failed to register route", "err", err)
		return err
	} else {
		log.Info(r, "Registered route", "prefix", prefix)
	}

	return nil
}

func (r *RepoSvs) unregisterRoutes(prefix enc.Name) (err error) {
	if _, err = r.client.Engine().ExecMgmtCmd("rib", "unregister", &mgmt.ControlArgs{
		Name:   prefix,
		Origin: optional.Some(uint64(mgmt.RouteOriginClient)),
	}); err != nil {
		log.Error(r, "Failed to unregister route", "err", err)
		return err
	} else {
		log.Info(r, "Unregistered route", "prefix", prefix)
	}

	return nil
}
