package repo

import (
	"fmt"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
)

type RepoSvs struct {
	config *Config
	group  enc.Name
	client ndn.Client
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

func (r *RepoSvs) Start() error {
	log.Info(r, "Starting SVS persistence")
	return nil
}

func (r *RepoSvs) Stop() error {
	log.Info(r, "Stopping SVS persistence")
	return nil
}
