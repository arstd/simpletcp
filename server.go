package simpletcp

import (
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/arstd/log"
)

const ReadBufferSize = 1024 * 1024
const WriteBufferSize = 1024 * 1024
const Processors = 8

type Server struct {
	Host string
	Port int

	ReadBufferSize  int // read buffer size of one connection
	WriteBufferSize int // write buffer size of one connection

	QueueSize  int // frame queue in/out size
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
	if s.WriteBufferSize == 0 {
		s.WriteBufferSize = WriteBufferSize
	}
	if s.ReadBufferSize == 0 {
		s.ReadBufferSize = ReadBufferSize
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

	// conn.SetKeepAlive(true)
	// conn.SetKeepAlivePeriod(10 * time.Second)

	inQueue := make(chan *Frame, s.QueueSize)
	outQueue := make(chan *Frame, s.QueueSize)

	go s.ReadLoop(inQueue, conn)

	for i := 0; i < s.Processors; i++ {
		if s.HandleFrame != nil {
			go s.handleFrameLoop(outQueue, inQueue)
		} else {
			go s.handleLoop(outQueue, inQueue)
		}
	}

	s.WriteLoop(outQueue, conn)
}

func (s *Server) ReadLoop(inQueue chan<- *Frame, conn *net.TCPConn) (err error) {
	defer conn.CloseRead()

	size := 4 * 1024 * 1024
	conn.SetReadBuffer(size)
	buf := make([]byte, size)

	f := NewFrameHead() // an uncomplete frame
	var head, body int  // readed head, readed body
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				return nil
			}

			if ne, ok := err.(net.Error); !ok || !ne.Timeout() {
				log.Error(err)
				return err
			}
		}

		if n == 0 { // no data
			continue
		}

		var i int
		for {
			// read head
			if head < HeadLength { // require head
				m := copy(f.head[head:], buf[i:n])
				head += m      // head required length
				i += m         // data from i: buf[i:n]
				if head >= 2 { // fixed head complete at least
					fh := f.FixedHead()
					if fh[0] != Fixed[0] || fh[1] != Fixed[1] {
						log.Error(ErrFixedHead)
						return ErrFixedHead
					}
				}

				if head < HeadLength { // data not enough to head
					break
				}
				body = 0
				f.data = make([]byte, f.DataLength())
			}

			// read body
			m := copy(f.data[body:], buf[i:n])
			body += m // body required length
			i += m    // data from i: buf[i:n]

			if body < f.DataLength() { // data not enough to body
				break
			}
			// frame complete

			inQueue <- f

			// another frame
			head = 0
			f = NewFrameHead()

			if i >= n {
				break
			}
		}
	}
}

func (s *Server) handleFrameLoop(outQueue chan<- *Frame, inQueue <-chan *Frame) (err error) {
	defer func() {
		if err := recover(); err != nil {
			log.Stack(err)
			go s.handleFrameLoop(outQueue, inQueue)
		}
	}()
	var f *Frame
	for {
		f = <-inQueue
		if f.Version() != VersionPing {
			f = s.HandleFrame(f)
		}
		outQueue <- f
	}
}

func (s *Server) handleLoop(outQueue chan<- *Frame, inQueue <-chan *Frame) (err error) {
	defer func() {
		if err := recover(); err != nil {
			log.Stack(err)
			go s.handleLoop(outQueue, inQueue)
		}
	}()
	var f *Frame
	for {
		f = <-inQueue
		if f.Version() != VersionPing {
			f.SetDataWithLength(s.Handle(f.data))
		}
		outQueue <- f
	}
}

func (s *Server) WriteLoop(outQueue <-chan *Frame, conn *net.TCPConn) (err error) {
	defer conn.CloseWrite()

	size := 1024 * 1024
	conn.SetWriteBuffer(size)
	buf := make([]byte, size)
	var i int
	var f *Frame
	for {
		select {
		case f = <-outQueue:
			if i+HeadLength+f.DataLength() > size {
				if _, err := conn.Write(buf[:i]); err != nil {
					log.Error(err)
					return err
				}
				i = 0
			}
			i += copy(buf[i:], f.head)
			i += copy(buf[i:], f.data)

		default:
			if i == 0 { // buf no data
				time.Sleep(100 * time.Microsecond)
				break
			}
			if i > 0 { // buf not full but no frame
				if _, err := conn.Write(buf[:i]); err != nil {
					log.Error(err)
					return err
				}
				i = 0
			}
		}
	}
}
