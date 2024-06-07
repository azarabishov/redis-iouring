package iomultiplexer

import (
	"fmt"
	"time"

	"github.com/iceber/iouring-go"
)

var readSize = 1024
var (
	iour     *IOUring
	Resulter chan iouring.Result
)


type IOUring struct {
	m *iouring.IOURing
}

func New(maxClients int) (*IOUring, error) {
	if maxClients < 0 {
		return nil, ErrInvalidMaxClients
	}
	Resulter = make(chan iouring.Result, 10)

	m, err := iouring.New(1024)
	if err != nil {
		return nil, err
	}
	
	iour = &IOUring{
		m:          m,
	}
	return iour, nil
}

func (s *IOUring) SubmitAccept(fd int) (error) {
	if _, err := s.m.SubmitRequest(iouring.Accept(fd), Resulter); err != nil {
		return fmt.Errorf("epoll subscribe: %w", err)
	}
	return nil
} 


func (s *IOUring) SubmitRead(fd int) (error) {
	buffer := make([]byte, readSize)
	prep := iouring.Read(fd, buffer)
	if _, err := iour.m.SubmitRequest(prep, Resulter); err != nil {
		return err
	}
	return nil
} 

func Write(fd int, data []byte) {
	prep := iouring.Write(fd, data)
	if _, err := iour.m.SubmitRequest(prep, Resulter); err != nil {
		return
	}
}


func (i *IOUring) Poll(timeout time.Duration) (Event, error) {
	for {
		result := <-Resulter
		switch result.Opcode() {
		case iouring.OpAccept:
			return Event{
				Fd: result.ReturnValue0().(int),
				Op: uint32(iouring.OpAccept),
			}, nil
		case iouring.OpRead:
			num := result.ReturnValue0().(int)
			buf, _ := result.GetRequestBuffer()
			content := buf[:num]
			return Event{
				Fd: result.Fd(),
				Data: content,
				Op: uint32(iouring.OpRead),
			}, nil
		}
	}
}


func (s *IOUring) Close() error {
	s.m.Close()
	return nil
}