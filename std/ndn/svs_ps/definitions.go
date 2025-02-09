//go:generate gondn_tlv_gen
package svs_ps

import enc "github.com/named-data/ndnd/std/encoding"

// +tlv-model:nocopy
type HistorySnap struct {
	//+field:sequence:*HistorySnapEntry:struct:HistorySnapEntry:nocopy
	Entries []*HistorySnapEntry `tlv:"0x82"`
}

// +tlv-model:nocopy
type HistorySnapEntry struct {
	//+field:natural
	SeqNo uint64 `tlv:"0xd6"`
	//+field:wire
	Content enc.Wire `tlv:"0x83"`
}

// +tlv-model:nocopy
type HistoryIndex struct {
	//+field:sequence:uint64:natural
	SeqNos []uint64 `tlv:"0x84"`
}
