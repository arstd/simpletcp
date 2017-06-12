package main

import (
	"bytes"
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/arstd/simpletcp"
)

func process(f *simpletcp.Frame) error {
	// log.Printf("read: %+v", f)

	f.Data = bytes.ToUpper(f.Data)

	// log.Printf("write: %+v", f)
	return nil
}

func main() {
	go http.ListenAndServe("0.0.0.0:6060", nil)

	server := &simpletcp.Server{
		Host: "0.0.0.0",
		Port: 8623,

		// FixedHeader : simpletcp.FixedHeader,
		// Version   : simpletcp.Version1,
		// DataType: simpletcp.DataTypeJSON,
		// MaxLength: simpletcp.MaxLength,

		HandleFunc: process,
	}

	log.Printf("server is running at %s:%d", server.Host, server.Port)
	log.Fatal(server.Start())
}
