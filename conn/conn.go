package conn

type Conn interface {
	SendFace(face uint64)
}
