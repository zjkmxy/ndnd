package repo

import (
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/engine"
	"github.com/named-data/ndnd/std/engine/basic"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/object"
)

type Repo struct {
	config *Config

	engine ndn.Engine
	store  ndn.Store
	client ndn.Client

	groupsSvs map[string]*RepoSvs
}

func NewRepo(config *Config) *Repo {
	return &Repo{
		config:    config,
		groupsSvs: make(map[string]*RepoSvs),
	}
}

func (r *Repo) String() string {
	return "repo"
}

func (r *Repo) Start() (err error) {
	log.Info(r, "Starting NDN Data Repository", "dir", r.config.StorageDir)

	// Make object store database
	r.store, err = object.NewBadgerStore(r.config.StorageDir + "/badger")
	if err != nil {
		return err
	}

	// Create NDN engine
	r.engine = engine.NewBasicEngine(engine.NewDefaultFace())
	r.setupEngineHook()
	if err = r.engine.Start(); err != nil {
		return err
	}

	// TODO: trust configuration
	r.client = object.NewClient(r.engine, r.store, nil)
	if err := r.client.Start(); err != nil {
		return err
	}

	// TODO: register Repo command prefix and handlers

	// Start test group (TODO: remove)
	test, _ := enc.NameFromStr("/ndn/svs")
	if err := r.startSvs(test); err != nil {
		log.Error(nil, "Failed to start test group", "err", err)
	}

	return nil
}

func (r *Repo) Stop() error {
	log.Info(r, "Stopping NDN Data Repository")

	for _, svs := range r.groupsSvs {
		svs.Stop()
	}
	clear(r.groupsSvs)

	if r.client != nil {
		r.client.Stop()
	}
	if r.engine != nil {
		r.engine.Stop()
	}

	return nil
}

func (r *Repo) startSvs(group enc.Name) error {
	// Check if already started
	if _, ok := r.groupsSvs[group.String()]; ok {
		return nil
	}

	// Start group
	svs := NewRepoSvs(r.config, group, r.client)
	if err := svs.Start(); err != nil {
		return err
	}
	r.groupsSvs[group.String()] = svs

	return nil
}

// setupEngineHook sets up the hook to persist all data.
func (r *Repo) setupEngineHook() {
	r.engine.(*basic.Engine).OnDataHook = func(data ndn.Data, raw enc.Wire, sigCov enc.Wire) error {
		// This is very hacky, improve if possible.
		// Assume that if there is a version it is the second-last component.
		// We might not want to store non-versioned data anyway (?)
		if ver := data.Name().At(-2); ver.IsVersion() {
			return r.store.Put(data.Name(), ver.NumberVal(), raw.Join())
		}
		return nil
	}
}
