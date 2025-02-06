package table

import (
	"github.com/named-data/ndnd/dv/config"
	"github.com/named-data/ndnd/dv/tlv"
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
)

type PrefixTable struct {
	config  *config.Config
	publish func(enc.Wire)
	routers map[string]*PrefixTableRouter
	me      *PrefixTableRouter
}

type PrefixTableRouter struct {
	Prefixes map[string]*PrefixEntry
}

type PrefixEntry struct {
	Name enc.Name
}

func NewPrefixTable(config *config.Config, publish func(enc.Wire)) *PrefixTable {
	pt := &PrefixTable{
		config:  config,
		publish: publish,
		routers: make(map[string]*PrefixTableRouter),
		me:      nil,
	}
	pt.me = pt.GetRouter(config.RouterName())
	return pt
}

func (pt *PrefixTable) String() string {
	return "dv-prefix"
}

func (pt *PrefixTable) GetRouter(name enc.Name) *PrefixTableRouter {
	hash := name.TlvStr()
	router := pt.routers[hash]
	if router == nil {
		router = &PrefixTableRouter{
			Prefixes: make(map[string]*PrefixEntry),
		}
		pt.routers[hash] = router
	}
	return router
}

func (pt *PrefixTable) Reset() {
	log.Info(pt, "Reset table")
	clear(pt.me.Prefixes)

	op := tlv.PrefixOpList{
		ExitRouter:    &tlv.Destination{Name: pt.config.RouterName()},
		PrefixOpReset: true,
	}
	pt.publish(op.Encode())
}

func (pt *PrefixTable) Announce(name enc.Name) {
	log.Info(pt, "Announce prefix", "name", name)
	hash := name.TlvStr()

	// Skip if matching entry already exists
	// This will also need to check that all params are equal
	if entry := pt.me.Prefixes[hash]; entry != nil {
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
	pt.publish(op.Encode())
}

func (pt *PrefixTable) Withdraw(name enc.Name) {
	log.Info(pt, "Withdraw prefix", "name", name)
	hash := name.TlvStr()

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
	pt.publish(op.Encode())
}

// Applies ops from a list. Returns if dirty.
func (pt *PrefixTable) Apply(wire enc.Wire) (dirty bool) {
	ops, err := tlv.ParsePrefixOpList(enc.NewWireView(wire), true)
	if err != nil {
		log.Warn(pt, "Failed to parse PrefixOpList", "err", err)
		return false
	}

	if ops.ExitRouter == nil || len(ops.ExitRouter.Name) == 0 {
		log.Error(pt, "Received PrefixOpList has no ExitRouter")
		return false
	}

	router := pt.GetRouter(ops.ExitRouter.Name)

	if ops.PrefixOpReset {
		log.Info(pt, "Reset remote prefixes", "router", ops.ExitRouter.Name)
		router.Prefixes = make(map[string]*PrefixEntry)
		dirty = true
	}

	for _, add := range ops.PrefixOpAdds {
		log.Info(pt, "Add remote prefix", "router", ops.ExitRouter.Name, "name", add.Name)
		router.Prefixes[add.Name.TlvStr()] = &PrefixEntry{Name: add.Name}
		dirty = true
	}

	for _, remove := range ops.PrefixOpRemoves {
		log.Info(pt, "Remove remote prefix", "router", ops.ExitRouter.Name, "name", remove.Name)
		delete(router.Prefixes, remove.Name.TlvStr())
		dirty = true
	}

	return dirty
}

func (pt *PrefixTable) Snap() enc.Wire {
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

	return snap.Encode()
}
