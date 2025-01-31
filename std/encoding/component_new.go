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

func NewGenericComponent(val string) Component {
	return NewStringComponent(TypeGenericNameComponent, val)
}

func NewGenericBytesComponent(val []byte) Component {
	return NewBytesComponent(TypeGenericNameComponent, val)
}

func NewKeywordComponent(val string) Component {
	return NewStringComponent(TypeKeywordNameComponent, val)
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

func (c Component) IsGeneric(text string) bool {
	return c.Typ == TypeGenericNameComponent && string(c.Val) == text
}

func (c Component) IsKeyword(keyword string) bool {
	return c.Typ == TypeKeywordNameComponent && string(c.Val) == keyword
}

func (c Component) IsSegment() bool {
	return c.Typ == TypeSegmentNameComponent
}

func (c Component) IsByteOffset() bool {
	return c.Typ == TypeByteOffsetNameComponent
}

func (c Component) IsSequenceNum() bool {
	return c.Typ == TypeSequenceNumNameComponent
}

func (c Component) IsVersion() bool {
	return c.Typ == TypeVersionNameComponent
}

func (c Component) IsTimestamp() bool {
	return c.Typ == TypeTimestampNameComponent
}
