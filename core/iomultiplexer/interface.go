package iomultiplexer

import "time"

type IOMultiplexer interface {
	SubmitAccept(fd int) (error)
	SubmitRead(fd int) (error)

	Poll(timeout time.Duration) (Event, error)
	Close() error
}
