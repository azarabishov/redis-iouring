package iomultiplexer

type Event struct {
	Fd int
	Data []byte
	Op uint32
}

type Operations uint32
 