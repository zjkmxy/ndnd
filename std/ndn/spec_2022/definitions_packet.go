package spec_2022

// +tlv-model:nocopy,private
type Packet struct {
	//+field:struct:Interest:nocopy
	Interest *Interest `tlv:"0x05"`
	//+field:struct:Data:nocopy
	Data *Data `tlv:"0x06"`
	//+field:struct:LpPacket:nocopy
	LpPacket *LpPacket `tlv:"0x64"`
}
