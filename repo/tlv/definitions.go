//go:generate gondn_tlv_gen
package tlv

import enc "github.com/named-data/ndnd/std/encoding"

var SyncProtocolSvsV3 = enc.Name{
	enc.NewKeywordComponent("ndn"),
	enc.NewKeywordComponent("svs"),
	enc.NewVersionComponent(3),
}

type RepoCmd struct {
	//+field:struct:SyncJoin
	SyncJoin *SyncJoin `tlv:"0x190"`
}

type RepoCmdRes struct {
	//+field:natural
	Status uint64 `tlv:"0x291"`
	//+field:string
	Message string `tlv:"0x292"`
}

type SyncJoin struct {
	//+field:name
	Protocol enc.Name `tlv:"0x191"`
	//+field:name
	Group enc.Name `tlv:"0x193"`
	//+field:struct:HistorySnapshotConfig
	HistorySnapshot *HistorySnapshotConfig `tlv:"0x1A4"`
}

type HistorySnapshotConfig struct {
	//+field:natural
	Threshold uint64 `tlv:"0x1A5"`
}
