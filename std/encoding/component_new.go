package encoding

func NewBytesComponent(typ TLNum, val []byte) Component {
	return Component{
		Typ: typ,
		Val: val,
	}
}

func NewStringComponent(typ TLNum, val string) Component {
	return Component{
		Typ: typ,
		Val: []byte(val),
	}
}

func NewNumberComponent(typ TLNum, val uint64) Component {
	return Component{
		Typ: typ,
		Val: Nat(val).Bytes(),
	}
}

func NewSegmentComponent(seg uint64) Component {
	return NewNumberComponent(TypeSegmentNameComponent, seg)
}

func NewByteOffsetComponent(off uint64) Component {
	return NewNumberComponent(TypeByteOffsetNameComponent, off)
}

func NewSequenceNumComponent(seq uint64) Component {
	return NewNumberComponent(TypeSequenceNumNameComponent, seq)
}

func NewVersionComponent(v uint64) Component {
	return NewNumberComponent(TypeVersionNameComponent, v)
}

func NewTimestampComponent(t uint64) Component {
	return NewNumberComponent(TypeTimestampNameComponent, t)
}
