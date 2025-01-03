package table

import (
	"github.com/named-data/ndnd/dv/config"
	"github.com/named-data/ndnd/dv/tlv"
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/object"
	ndn_sync "github.com/named-data/ndnd/std/sync"
	"github.com/named-data/ndnd/std/utils"
)

const PrefixTableSnapThreshold = 100

var PrefixTableSnap = enc.NewStringComponent(enc.TypeKeywordNameComponent, "SNAP")

type PrefixTable struct {
	config *config.Config
	client *object.Client
	svs    *ndn_sync.SvSync

	routers map[uint64]*PrefixTableRouter
	me      *PrefixTableRouter

	snapshotAt uint64
}

type PrefixTableRouter struct {
	Name     enc.Name
	Fetching bool
	Known    uint64
	Latest   uint64
	Prefixes map[uint64]*PrefixEntry
}

type PrefixEntry struct {
	Name enc.Name
}

func NewPrefixTable(
	config *config.Config,
	client *object.Client,
	svs *ndn_sync.SvSync,
) *PrefixTable {
	pt := &PrefixTable{
		config: config,
		client: client,
		svs:    svs,

		routers: make(map[uint64]*PrefixTableRouter),
		me:      nil,
	}

	pt.me = pt.GetRouter(config.RouterName())
	pt.me.Known = svs.GetSeqNo(config.RouterName())
	pt.me.Latest = pt.me.Known
	pt.publishSnap()

	return pt
}

func (pt *PrefixTable) GetRouter(name enc.Name) *PrefixTableRouter {
	hash := name.Hash()
	router := pt.routers[hash]
	if router == nil {
		router = &PrefixTableRouter{
			Name:     name,
			Prefixes: make(map[uint64]*PrefixEntry),
		}
		pt.routers[hash] = router
	}
	return router
}

func (pt *PrefixTable) Announce(name enc.Name) {
	log.Infof("prefix-table: announcing %s", name)
	hash := name.Hash()

	// Skip if matching entry already exists
	// This will also need to check that all params are equal
	if entry := pt.me.Prefixes[hash]; entry != nil && entry.Name.Equal(name) {
		return
	}

	// Create new entry and announce globally
	pt.me.Prefixes[hash] = &PrefixEntry{Name: name}

	op := tlv.PrefixOpList{
		ExitRouter: &tlv.Destination{Name: pt.config.RouterName()},
		PrefixOpAdds: []*tlv.PrefixOpAdd{{
			Name: name,
			Cost: 1,
		}},
	}
	pt.publishOp(op.Encode())
}

func (pt *PrefixTable) Withdraw(name enc.Name) {
	log.Infof("prefix-table: withdrawing %s", name)
	hash := name.Hash()

	// Check if entry does not exist
	if entry := pt.me.Prefixes[hash]; entry == nil {
		return
	}

	// Delete the existing entry and announce globally
	delete(pt.me.Prefixes, hash)

	op := tlv.PrefixOpList{
		ExitRouter:      &tlv.Destination{Name: pt.config.RouterName()},
		PrefixOpRemoves: []*tlv.PrefixOpRemove{{Name: name}},
	}
	pt.publishOp(op.Encode())
}

// Applies ops from a list. Returns if dirty.
func (pt *PrefixTable) Apply(ops *tlv.PrefixOpList) (dirty bool) {
	if ops.ExitRouter == nil || len(ops.ExitRouter.Name) == 0 {
		log.Error("prefix-table: received PrefixOpList has no ExitRouter")
		return false
	}

	router := pt.GetRouter(ops.ExitRouter.Name)

	if ops.PrefixOpReset {
		log.Infof("prefix-table: reset prefix table for %s", ops.ExitRouter.Name)
		router.Prefixes = make(map[uint64]*PrefixEntry)
		dirty = true
	}

	for _, add := range ops.PrefixOpAdds {
		log.Infof("prefix-table: added prefix for %s: %s", ops.ExitRouter.Name, add.Name)
		router.Prefixes[add.Name.Hash()] = &PrefixEntry{Name: add.Name}
		dirty = true
	}

	for _, remove := range ops.PrefixOpRemoves {
		log.Infof("prefix-table: removed prefix for %s: %s", ops.ExitRouter.Name, remove.Name)
		delete(router.Prefixes, remove.Name.Hash())
		dirty = true
	}

	return dirty
}

func (pt *PrefixTable) publishOp(content enc.Wire) {
	// Increment our sequence number
	seq := pt.svs.IncrSeqNo(pt.config.RouterName())
	pt.me.Known = seq
	pt.me.Latest = seq

	// Produce the operation
	_, err := pt.client.Produce(object.ProduceArgs{
		Name:    append(pt.config.PrefixTableDataPrefix(), enc.NewSequenceNumComponent(seq)),
		Content: content,
		Version: utils.IdPtr(uint64(0)), // immutable
	})
	if err != nil {
		log.Errorf("prefix-table: failed to produce op: %v", err)
		return
	}

	// Create snapshot if needed
	if seq-pt.snapshotAt >= PrefixTableSnapThreshold/2 {
		pt.publishSnap()
	}
}

func (pt *PrefixTable) publishSnap() {
	snap := tlv.PrefixOpList{
		ExitRouter:    &tlv.Destination{Name: pt.config.RouterName()},
		PrefixOpReset: true,
		PrefixOpAdds:  make([]*tlv.PrefixOpAdd, 0, len(pt.me.Prefixes)),
	}

	for _, entry := range pt.me.Prefixes {
		snap.PrefixOpAdds = append(snap.PrefixOpAdds, &tlv.PrefixOpAdd{
			Name: entry.Name,
			Cost: 1,
		})
	}

	_, err := pt.client.Produce(object.ProduceArgs{
		Name:    append(pt.config.PrefixTableDataPrefix(), PrefixTableSnap),
		Content: snap.Encode(),
		Version: utils.IdPtr(pt.me.Latest),
	})
	if err != nil {
		log.Errorf("prefix-table: failed to produce snap: %v", err)
	}

	pt.snapshotAt = pt.me.Latest
}
