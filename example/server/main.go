package main

import (
	"bytes"

	"github.com/arstd/log"
	"github.com/arstd/simpletcp"
)

func handle(data []byte) []byte {
	log.Debug(data)
	return bytes.ToUpper(data)
}

func main() {
	srv := &simpletcp.Server{
		Host: "",
		Port: 8090,

		QueueSize:  32,
		Processors: 8,

		Handle: func(data []byte) []byte { return data },
	}
	log.Printf("tcp server is listening at %s:%d", srv.Host, srv.Port)
	log.Fataln(srv.Start())
}
