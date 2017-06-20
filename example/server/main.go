package main

import (
	"bytes"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arstd/log"
	"github.com/arstd/simpletcp/simple"
	"github.com/arstd/tcp"
)

func handle(data []byte) []byte {
	log.Debug(data)
	return bytes.ToUpper(data)
}

func main() {
	// creates a server
	config := &tcp.Config{
		Addr: ":8623",
		CallbackProcessorLimit: 64,
		PacketSendChanLimit:    20,
		PacketReceiveChanLimit: 20,
	}
	srv := tcp.NewServer(config, simple.Callback(handle), &simple.Protocol{})

	go func() {
		log.Printf("tcp server is listening at %s", config.Addr)
		log.Errorn(srv.Start(time.Second))
	}()

	// wait signal to shutdown gracefully
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL, syscall.SIGHUP, syscall.SIGQUIT)
	log.Printf("receive signal `%s`", <-sig)

	log.Print("stop tcp server gracefully")
	srv.Stop()
}
