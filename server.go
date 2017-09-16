package simpletcp

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/arstd/log"
)

const (
	QueueSize  = 4096
	Processors = 32
)

type Server struct {
	Host string
	Port int

	Fixed     [2]byte // default 'Ac' (0x41 0x63)
	MaxLength uint32  // default 65536 (1<<16)

	Version  byte // default 1 (0x01)
	BodyType byte // default 1 (0x01, json)

	Handle      func([]byte) []byte // one of handlers must not nil
	HandleFrame func(*Frame) *Frame

	l    *net.TCPListener
	exit chan struct{}  // send exit signal
	wg   sync.WaitGroup // wait all conns to exit, then stop server
}

func (s *Server) init() error {
	if s.Handle == nil && s.HandleFrame == nil {
		return errors.New("one of handle/handleFrame func not nil")
	}

	if s.Host == "" {
		s.Host = "0.0.0.0"
	}
	if s.Fixed == [2]byte{} {
		s.Fixed = Fixed
	}
	if s.Version == 0 {
		s.Version = Version1
	}
	if s.BodyType == 0 {
		s.BodyType = BodyTypeJSON
	}
	if s.MaxLength == 0 {
		s.MaxLength = MaxLength
	}

	s.exit = make(chan struct{})

	return nil
}

func (s *Server) Start() (err error) {
	if err = s.init(); err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}
	s.l, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}

	for {
		conn, err := s.l.AcceptTCP()
		if err != nil {
			select {
			case <-s.exit:
				return nil
			default:
				log.Error(err)
				return err
			}
		}
		go s.process(conn)
	}
}

func (s *Server) process(conn *net.TCPConn) {
	s.wg.Add(1)

	log.Infof("accept connection from %s", conn.RemoteAddr())

	c := NewConnect(conn, s)
	c.Run()

	s.wg.Done()
}

func (s *Server) Close() error {
	log.Trace("send server exit signal")
	close(s.exit) // send exit signal

	s.wg.Wait()
	log.Trace("all connection exit")

	log.Trace("exit server")
	return s.l.Close()
}
