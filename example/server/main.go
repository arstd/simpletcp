package main

import (
	"bytes"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/arstd/log"
	"github.com/arstd/simpletcp"
)

func handle(data []byte) []byte {
	log.Debug(data)
	return bytes.ToUpper(data)
}

func main() {
	var exit = make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go runTCP(exit, &wg)

	go await(exit)

	wg.Wait()
	log.Debug("all async goroutine exit normally")
}

func runTCP(exit chan struct{}, wg *sync.WaitGroup) {
	srv := &simpletcp.Server{
		Host: "",
		Port: 8090,

		QueueSize:  32,
		Processors: 1,

		Handle: func(data []byte) []byte { return data },
	}
	go func() {
		log.Printf("tcp server is listening at %s:%d", srv.Host, srv.Port)
		log.Fataln(srv.Start())
	}()

	<-exit

	log.Print("graceful close tcp server")
	log.Fataln(srv.Close())
	wg.Done()
}

func await(exit chan struct{}) {
	// wait signal to shutdown gracefully
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL, syscall.SIGHUP, syscall.SIGQUIT)
	log.Printf("receive signal `%s`", <-sig)

	close(exit)
}
