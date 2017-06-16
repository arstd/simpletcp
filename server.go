package simpletcp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/arstd/log"
)

const BufferSize = 20480
const Processors = 32

type Server struct {
	Host string
	Port int

	BufferSize int // read and write buffer size of one connection
	Processors int // goroutine number of one connection

	Fixed     [2]byte // default 'Ac' (0x41 0x63)
	MaxLength uint32  // default 65536 (1<<16)

	Version  byte // default 1 (0x01)
	DataType byte // default 1 (0x01, json)

	Handle      func([]byte) []byte // one of handlers must not nil
	HandleFrame func(*Frame) *Frame
}

func (s *Server) init() error {
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
	if s.BufferSize == 0 {
		s.BufferSize = BufferSize
	}
	if s.Processors == 0 {
		s.Processors = Processors
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
	// defer conn.Close()

	conn.SetNoDelay(true)

	conn.SetKeepAlive(true)
	conn.SetKeepAlivePeriod(10 * time.Second)

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
	conn.SetReadBuffer(2048)
	br := bufio.NewReaderSize(conn, 2048)
	for {
		if frame, err := Read(br, s.Fixed, s.MaxLength); err != nil {
			if err == io.EOF {
				conn.Close()
			} else {
				log.Error(err)
				conn.CloseRead()
			}
			return err
		} else {
			inQueue <- frame
		}
	}
}

func (s *Server) processLoopFrame(outQueue chan<- *Frame, inQueue <-chan *Frame) (err error) {
	defer func() {
		if err := recover(); err != nil {
			log.Stack(err)
			go s.processLoopFrame(outQueue, inQueue)
		}
	}()
	var frame *Frame
	for {
		frame = <-inQueue
		if frame.Version == VersionPing {
			outQueue <- frame
		} else {
			outQueue <- s.HandleFrame(frame)
		}
	}
}

func (s *Server) processLoop(outQueue chan<- *Frame, inQueue <-chan *Frame) (err error) {
	defer func() {
		if err := recover(); err != nil {
			log.Stack(err)
			go s.processLoop(outQueue, inQueue)
		}
	}()
	var frame *Frame
	for {
		frame = <-inQueue
		if frame.Version == VersionPing {
			outQueue <- frame
		} else {
			frame.Data = s.Handle(frame.Data)
			outQueue <- frame
		}
	}
}

func (s *Server) writeLoop(outQueue <-chan *Frame, conn *net.TCPConn) (err error) {
	conn.SetWriteBuffer(1024)
	bw := bufio.NewWriterSize(conn, 1024)
	for {
		frame := <-outQueue
		if err = Write(bw, s.Fixed, frame); err != nil {
			if err == io.EOF {
				conn.Close()
			} else {
				log.Error(err)
				conn.CloseWrite()
			}
			return
		}
	}
}
