package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"redis-epoll/config"
	"redis-epoll/server"
)

func setupFlags() {
	flag.StringVar(&config.Host, "host", "0.0.0.0", "host for the server")
	flag.IntVar(&config.Port, "port", 7379, "port for the server")
	flag.Parse()
}

func main() {
	setupFlags()
	log.Println("server started \u2684\uFE0E")

	var sigs chan os.Signal = make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	var wg sync.WaitGroup
	wg.Add(1)

	go server.RunAsyncTCPServer(&wg)
	go server.WaitForSignal(&wg, sigs)

	wg.Wait()
}
