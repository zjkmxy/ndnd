package repo

import (
	"fmt"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
	spec_svs "github.com/named-data/ndnd/std/ndn/svs/v3"
	ndn_sync "github.com/named-data/ndnd/std/sync"
	"github.com/named-data/ndnd/std/types/optional"
)

type RepoSvs struct {
	config *Config
	group  enc.Name
	client ndn.Client

	svsalo *ndn_sync.SvsALO
	routes []enc.Name
}

func NewRepoSvs(config *Config, group enc.Name, client ndn.Client) *RepoSvs {
	return &RepoSvs{
		config: config,
		group:  group,
		client: client,
	}
}

func (r *RepoSvs) String() string {
	return fmt.Sprintf("repo-svs (%s)", r.group)
}

func (r *RepoSvs) Start() (err error) {
	log.Info(r, "Starting SVS persistence")

	// Get previous saved state
	initialState := r.readState()

	r.svsalo, err = ndn_sync.NewSvsALO(ndn_sync.SvsAloOpts{
		Name:         enc.Name{enc.NewKeywordComponent("repo")}, // unused,
		InitialState: initialState,
		Svs: ndn_sync.SvSyncOpts{
			Client:      r.client,
			GroupPrefix: r.group,
			Passive:     true,
		},

		// TODO: support other snapshot strategies
		// TODO: force fetch all snapshots
		Snapshot: &ndn_sync.SnapshotNodeHistory{
			Client:    r.client,
			Threshold: 10, // TODO: depends on app
			IsRepo:    true,
		},
	})
	if err != nil {
		return err
	}

	r.svsalo.SetOnError(func(err error) {
		log.Error(r, "SVS ALO error", "err", err)
	})

	r.svsalo.SubscribePublisher(enc.Name{}, func(pub ndn_sync.SvsPub) {
		r.commitState(pub.State)
		r.registerPublisherRoute(pub.Publisher)
	})

	if err = r.registerRoute(r.svsalo.SyncPrefix()); err != nil {
		return err
	}

	if err = r.svsalo.Start(); err != nil {
		return err
	}

	// Register prefixes from existing members
	r.processInitialState(initialState)

	return nil
}

func (r *RepoSvs) Stop() (err error) {
	log.Info(r, "Stopping SVS persistence")

	if r.svsalo == nil {
		return nil
	}

	r.unregisterRoutes()

	if err = r.svsalo.Stop(); err != nil {
		return err
	}

	return nil
}

func (r *RepoSvs) commitState(state enc.Wire) {
	name := r.group.Append(enc.NewKeywordComponent("alo-state"))
	r.client.Store().Put(name, 0, state.Join())
}

func (r *RepoSvs) readState() enc.Wire {
	name := r.group.Append(enc.NewKeywordComponent("alo-state"))
	if stateWire, _ := r.client.Store().Get(name, false); stateWire != nil {
		return enc.Wire{stateWire}
	}
	return nil
}

func (r *RepoSvs) registerPublisherRoute(name enc.Name) error {
	// Register the route for the publisher without boot time
	return r.registerRoute(name.Append(r.group...))
}

func (r *RepoSvs) registerRoute(prefix enc.Name) (err error) {
	for _, reg := range r.routes {
		if reg.Equal(prefix) {
			return nil
		}
	}

	// Disable route inheritance
	if _, err = r.client.Engine().ExecMgmtCmd("rib", "register", &mgmt.ControlArgs{
		Name:  prefix,
		Mask:  optional.Some(uint64(mgmt.RouteFlagChildInherit)),
		Flags: optional.Some(uint64(0)),
	}); err != nil {
		log.Error(r, "Failed to register route", "err", err)
		return err
	} else {
		log.Info(r, "Registered route", "prefix", prefix)
	}

	r.routes = append(r.routes, prefix)

	return nil
}

func (r *RepoSvs) unregisterRoutes() (err error) {
	for _, name := range r.routes {
		if _, err = r.client.Engine().ExecMgmtCmd("rib", "unregister", &mgmt.ControlArgs{
			Name: name,
		}); err != nil {
			log.Error(r, "Failed to unregister route", "err", err)
			return err
		} else {
			log.Info(r, "Unregistered route", "prefix", name)
		}
	}
	r.routes = nil
	return nil
}

func (r *RepoSvs) processInitialState(wire enc.Wire) {
	if wire == nil {
		return
	}

	state, err := spec_svs.ParseInstanceState(enc.NewWireView(wire), true)
	if err != nil {
		return
	}

	for _, entry := range state.StateVector.Entries {
		if len(entry.SeqNoEntries) == 0 {
			continue
		}

		if err = r.registerPublisherRoute(entry.Name); err != nil {
			continue
		}
	}
}
