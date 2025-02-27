package repo

import (
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/engine"
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

	// Create NDN engine
	r.engine = engine.NewBasicEngine(engine.NewDefaultFace())

	// Make object store database
	r.store, err = object.NewBoltStore(r.config.StorageDir + "/bolt.db")
	if err != nil {
		return err
	}

	// TODO: trust configuration
	r.client = object.NewClient(r.engine, r.store, nil)

	// TODO: register Repo command prefix and handlers

	// Start test group (TODO: remove)
	test, _ := enc.NameFromStr("/ndnd/svstest")
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
