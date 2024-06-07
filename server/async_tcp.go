package server

import (
	"io"
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"redis-epoll/config"
	"redis-epoll/core"
	"redis-epoll/core/iomultiplexer"

	"github.com/iceber/iouring-go"
)

const (
	SubscribeOperationType_ACCEPT int32 = 1 << 1
	SubscribeOperationType_READ int32 = 1 << 2
	SubscribeOperationType_WRITE int32 = 1 << 3
)

var cronFrequency time.Duration = 1 * time.Second
var lastCronExecTime time.Time = time.Now()

const EngineStatus_WAITING int32 = 1 << 1
const EngineStatus_BUSY int32 = 1 << 2
const EngineStatus_SHUTTING_DOWN int32 = 1 << 3
const EngineStatus_TRANSACTION int32 = 1 << 4

var eStatus int32 = EngineStatus_WAITING

var connectedClients map[int]*core.Client

var readSize = 1024

func init() {
	connectedClients = make(map[int]*core.Client)
}


func WaitForSignal(wg *sync.WaitGroup, sigs chan os.Signal) {
	defer wg.Done()
	<-sigs

	for atomic.LoadInt32(&eStatus) == EngineStatus_BUSY {
	}

	atomic.StoreInt32(&eStatus, EngineStatus_SHUTTING_DOWN)

	os.Exit(0)
}

func RunAsyncTCPServer(wg *sync.WaitGroup) error {
	defer wg.Done()
	defer func() {
		atomic.StoreInt32(&eStatus, EngineStatus_SHUTTING_DOWN)
	}()

	log.Println("starting an asynchronous TCP server on", config.Host, config.Port)


	maxClients := 20000

	serverFD, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return err
	}
	defer syscall.Close(serverFD)

	ip4 := net.ParseIP(config.Host)

	if err = syscall.Bind(serverFD, &syscall.SockaddrInet4{
		Port: config.Port,
		Addr: [4]byte{ip4[0], ip4[1], ip4[2], ip4[3]},
	}); err != nil {
		return err
	}

	if err = syscall.Listen(serverFD, maxClients); err != nil {
		return err
	}

	var multiplexer iomultiplexer.IOMultiplexer
	multiplexer, err = iomultiplexer.New(maxClients)
	if err != nil {
		log.Fatal(err)
	}
	defer multiplexer.Close()

	
	if err := multiplexer.SubmitAccept(serverFD); err != nil {
		return err
	}


	for atomic.LoadInt32(&eStatus) != EngineStatus_SHUTTING_DOWN {
		if time.Now().After(lastCronExecTime.Add(cronFrequency)) {
			core.DeleteExpiredKeys()
			lastCronExecTime = time.Now()
		}

		event, err := multiplexer.Poll(-1)
		if err != nil {
			continue
		}

		if !atomic.CompareAndSwapInt32(&eStatus, EngineStatus_WAITING, EngineStatus_BUSY) {
			switch eStatus {
			case EngineStatus_SHUTTING_DOWN:
				return nil
			}
		}

		switch event.Op {
		case uint32(iouring.OpAccept):
			connectedClients[event.Fd] = core.NewClient(event.Fd)
			multiplexer.SubmitAccept(serverFD)
			multiplexer.SubmitRead(event.Fd)
			// syscall.SetNonblock(event.Fd, true)
		case uint32(iouring.OpRead):
			multiplexer.SubmitRead(event.Fd)

			comm := connectedClients[event.Fd]
			comm.Data = event.Data
			if comm == nil {
				continue
			}
			cmds, hasABORT, err := readCommands(comm)

			if err != nil && err != io.EOF {
				syscall.Close(event.Fd)
				delete(connectedClients, event.Fd)
				continue
			}
			respond(cmds, comm)
			if hasABORT {
				return nil
			}
		}

		atomic.StoreInt32(&eStatus, EngineStatus_WAITING)
	}

	return nil
}