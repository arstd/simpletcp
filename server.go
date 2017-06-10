package simpletcp

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
)

type Server struct {
	Host string
	Port int

	FixedHeader [2]byte // default 'Ac' (0x41 0x63)
	Version     byte    // default 1 (0x01)
	DataType    byte    // default 1 (0x01, json)
	MaxLength   uint32  // default 655356 (1<<16)

	HandleFunc func(*Frame) error // get data from frame.Data, then put result data back
}

func (s *Server) Start() error {
	if s.Host == "" {
		s.Host = "0.0.0.0"
	}
	if s.FixedHeader == [2]byte{} {
		s.FixedHeader = FixedHeader
	}
	if s.Version == 0 {
		s.Version = Version1
	}
	if s.DataType == 0 {
		s.DataType = DataTypeJSON
	}
	if s.MaxLength == 0 {
		s.MaxLength = MaxLength
	}

	if s.HandleFunc == nil {
		return errors.New("handle func must not nil")
	}

	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer l.Close()

	s.accept(l)

	return errors.New("unreachable code, tcp accept exception?")
}

func (s *Server) accept(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}
		go s.process(conn)
	}
}

func (s *Server) process(conn net.Conn) {
	log.Printf("connection from %s", conn.RemoteAddr())

	defer conn.Close()

	// go s.keepalive()

	buf := make([]byte, s.MaxLength)
	frame := &Frame{
		Header: Header{
			FixedHeader: s.FixedHeader,
			Version:     s.Version,
			DataType:    s.DataType,
			MaxLength:   s.MaxLength,
		},
		Data: buf,
	}
	var err error
	for {
		frame.Data = buf
		err = frame.Read(conn)
		if err != nil {
			if err != io.EOF {
				log.Print(err)
			}
			return
		}

		err = s.HandleFunc(frame)
		if err != nil {
			return
		}

		if err = frame.Write(conn); err != nil {
			if err != io.EOF {
				log.Print(err)
			}
			return
		}
	}
}
