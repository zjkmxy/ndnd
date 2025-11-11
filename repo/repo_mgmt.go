package repo

import (
	"fmt"

	"github.com/named-data/ndnd/repo/tlv"
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
)

// (AI GENERATED DESCRIPTION): Parses a repository management command from the received wire and dispatches it to the sync‑join handler if present, otherwise logs a warning about an unknown command.
func (r *Repo) onMgmtCmd(_ enc.Name, wire enc.Wire, reply func(enc.Wire) error) {
	cmd, err := tlv.ParseRepoCmd(enc.NewWireView(wire), false)
	if err != nil {
		log.Warn(r, "Failed to parse management command", "err", err)
		return
	}

	if cmd.SyncJoin != nil {
		go r.handleSyncJoin(cmd.SyncJoin, reply)
		return
	}

	log.Warn(r, "Unknown management command received")
}

// (AI GENERATED DESCRIPTION): Handles a `SyncJoin` command by starting an SVS session when the protocol is `SyncProtocolSvsV3`, or returning an error status if the protocol is unknown or the session fails to start.
func (r *Repo) handleSyncJoin(cmd *tlv.SyncJoin, reply func(enc.Wire) error) {
	res := tlv.RepoCmdRes{Status: 200}

	if cmd.Protocol != nil && cmd.Protocol.Name.Equal(tlv.SyncProtocolSvsV3) {
		if err := r.startSvs(cmd); err != nil {
			res.Status = 500
			log.Error(r, "Failed to start SVS", "err", err)
		}
		reply(res.Encode())
		return
	}

	log.Warn(r, "Unknown sync protocol specified in command", "protocol", cmd.Protocol)
	res.Status = 400
	reply(res.Encode())
}

// (AI GENERATED DESCRIPTION): Starts a SyncJoin session for the specified group, initializing a new RepoSvs if one isn’t already active and storing it in the repository’s group session map.
func (r *Repo) startSvs(cmd *tlv.SyncJoin) error {
	if cmd.Group == nil || len(cmd.Group.Name) == 0 {
		return fmt.Errorf("missing group name")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if already started
	hash := cmd.Group.Name.TlvStr()
	if _, ok := r.groupsSvs[hash]; ok {
		return nil
	}

	// Start group
	svs := NewRepoSvs(r.config, r.client, cmd)
	if err := svs.Start(); err != nil {
		return err
	}
	r.groupsSvs[hash] = svs

	return nil
}
