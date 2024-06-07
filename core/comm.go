package core

import (
	"io"
	"redis-epoll/core/iomultiplexer"
)

type Client struct {
	io.ReadWriter
	Fd     int
	Data    []byte
	cqueue RedisCmds
}


func (c *Client) Read(b []byte) (int, error) {
	if len(c.Data) == 0 {
		return 0, io.EOF
	}

	n := copy(b, c.Data)
	c.Data = c.Data[n:]

	return n, nil

}

func (c *Client) Write(b []byte) (int, error) {
	iomultiplexer.Write(c.Fd, b)
	return 0, nil
}



func NewClient(fd int) *Client {
	return &Client{
		Fd:     fd,
		cqueue: make(RedisCmds, 0),
	}
}
