package main

import (
	"bytes"
	"net/http"
	_ "net/http/pprof"

	"github.com/arstd/log"
	"github.com/arstd/simpletcp"
)

func handle(data []byte) []byte {
	log.Info(data)
	return bytes.ToUpper(data)
}

var scount int

func handleFrame(frame *simpletcp.Frame) *simpletcp.Frame {
	scount++
	log.Infof("%d: %d %d %s", scount, frame.MessageId, frame.DataLength, frame.Data)
	frame.Data = bytes.ToUpper(frame.Data)
	return frame
}

func main() {
	log.SetLevel(log.Lwarn)

	go http.ListenAndServe("0.0.0.0:6060", nil)

	server := &simpletcp.Server{
		Host: "0.0.0.0",
		Port: 8623,

		// FixedHeader : simpletcp.FixedHeader,
		// Version   : simpletcp.Version1,
		// DataType: simpletcp.DataTypeJSON,
		// MaxLength: simpletcp.MaxLength,

		// Handle: handle,
		HandleFrame: handleFrame,
	}

	log.Printf("server is running at %s:%d", server.Host, server.Port)
	log.Fatal(server.Start())
}
