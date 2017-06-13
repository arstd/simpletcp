package simpletcp

import (
	"bufio"
	"errors"
	"fmt"
	"net"

	"github.com/arstd/log"
)

const BufferSize = 4096
const Processors = 10

type Server struct {
	Host string
	Port int

	BufferSize int // read and write buffer size of one connection

	FixedHeader [2]byte // default 'Ac' (0x41 0x63)
	Version     byte    // default 1 (0x01)
	DataType    byte    // default 1 (0x01, json)
	MaxLength   uint32  // default 655356 (1<<16)

	Handle      func([]byte) []byte // one of handlers must not nil
	HandleFrame func(*Frame) *Frame
}

func (s *Server) init() error {
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
	if s.BufferSize == 0 {
		s.BufferSize = BufferSize
	}

	if s.Handle == nil && s.HandleFrame == nil {
		return errors.New("one of handle/handleFrame func not nil")
	}

	return nil
}

func (s *Server) Start() error {
	if err := s.init(); err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}
	l, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}
	defer l.Close()

	s.accept(l)

	return errors.New("unreachable code, tcp accept exception?")
}

func (s *Server) accept(l *net.TCPListener) error {
	for {
		conn, err := l.AcceptTCP()
		if err != nil {
			continue
		}
		go s.process(conn)
	}
}

func (s *Server) process(conn *net.TCPConn) {
	log.Infof("connection from %s", conn.RemoteAddr())
	// go s.keepalive(conn)
	// defer conn.Close()

	inQueue := make(chan *Frame, s.BufferSize)
	outQueue := make(chan *Frame, s.BufferSize)

	go s.readLoop(inQueue, conn)

	for i := 0; i < Processors; i++ {
		if s.HandleFrame != nil {
			go s.processLoopFrame(outQueue, inQueue)
		} else {
			go s.processLoop(outQueue, inQueue)
		}
	}

	go s.writeLoop(outQueue, conn)
}

func (s *Server) readLoop(inQueue chan<- *Frame, conn *net.TCPConn) error {
	br := bufio.NewReader(conn)
	for {
		if frame, err := Read(br, s.FixedHeader, s.MaxLength); err != nil {
			return err
		} else {
			inQueue <- frame
		}
	}
}

func (s *Server) processLoopFrame(outQueue chan<- *Frame, inQueue <-chan *Frame) (err error) {
	var frame *Frame
	for {
		frame = <-inQueue
		outQueue <- s.HandleFrame(frame)
	}
}

func (s *Server) processLoop(outQueue chan<- *Frame, inQueue <-chan *Frame) (err error) {
	var frame *Frame
	for {
		frame = <-inQueue
		frame.Data = s.Handle(frame.Data)
		outQueue <- frame
	}
}

func (s *Server) writeLoop(outQueue <-chan *Frame, conn *net.TCPConn) (err error) {
	bw := bufio.NewWriter(conn)
	for {
		frame := <-outQueue
		if frame.DataLength > s.MaxLength {
			return ErrDataLengthExceed
		}
		if err = Write(bw, frame); err != nil {
			return
		}
	}
}
