package mgmtconn

type baseconn interface {
	Make(socketFile string)
	Send(face uint64)
	RunReceive()
	process(size int, buf []byte)
}
