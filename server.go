package simpletcp

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/arstd/log"
)

const (
	ReadBufferSize  = 4 * 1024 * 1024
	WriteBufferSize = 2 * 1024 * 1024
	QueueSize       = 4096
	Processors      = 32
)

type Server struct {
	Host string
	Port int

	Fixed     [2]byte // default 'Ac' (0x41 0x63)
	MaxLength uint32  // default 65536 (1<<16)

	Version  byte // default 1 (0x01)
	DataType byte // default 1 (0x01, json)

	ReadBufferSize  int // read buffer size of one connection
	WriteBufferSize int // write buffer size of one connection

	QueueSize  int // frame queue in/out size
	Processors int // goroutine number of one connection

	Handle      func([]byte) []byte // one of handlers must not nil
	HandleFrame func(*Frame) *Frame

	l     *net.TCPListener
	close chan struct{}  // send close signal
	wg    sync.WaitGroup // wait all conns to close, then stop server
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
	if s.DataType == 0 {
		s.DataType = DataTypeJSON
	}
	if s.MaxLength == 0 {
		s.MaxLength = MaxLength
	}

	if s.QueueSize == 0 {
		s.QueueSize = QueueSize
	}
	if s.WriteBufferSize == 0 {
		s.WriteBufferSize = WriteBufferSize
	}
	if s.ReadBufferSize == 0 {
		s.ReadBufferSize = ReadBufferSize
	}
	if s.Processors == 0 {
		s.Processors = Processors
	}

	s.close = make(chan struct{})

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
			case <-s.close:
				return nil
			default:
				log.Error(err)
				return err
			}
		}
		s.wg.Add(1)
		go s.process(conn)
	}
}

func (s *Server) process(conn *net.TCPConn) {
	log.Infof("accept connection from %s", conn.RemoteAddr())

	c := NewConnect(conn, int32(s.QueueSize), s.ReadBufferSize, s.WriteBufferSize, s.Handle, s.HandleFrame)

	c.Process(s.Processors, s.close)
	s.wg.Done()
}

func (s *Server) Close() error {
	log.Trace("send server close signal")
	close(s.close) // send close signal

	s.wg.Wait()
	log.Trace("all connection closed")

	log.Trace("close server")
	return s.l.Close()
}
