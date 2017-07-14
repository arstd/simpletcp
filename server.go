package simpletcp

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
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

func (s *Server) Close() error {
	log.Trace("send server close signal")
	close(s.close) // send close signal

	s.wg.Wait()
	log.Trace("all connection closed")

	log.Trace("close server")
	return s.l.Close()
}

func (s *Server) process(conn *net.TCPConn) {
	log.Infof("accept connection from %s", conn.RemoteAddr())

	conn.SetNoDelay(true)

	inQueue := make(chan *Frame, s.QueueSize)
	outQueue := make(chan *Frame, s.QueueSize)

	go readLoop(inQueue, conn)

	var wg sync.WaitGroup
	for i := 0; i < s.Processors; i++ {
		wg.Add(1)
		if s.HandleFrame != nil {
			go handleFrameLoop(outQueue, inQueue, s.HandleFrame, &wg)
		} else {
			go handleLoop(outQueue, inQueue, s.Handle, &wg)
		}
	}

	closed := make(chan struct{}) // can read after connect closed
	go writeLoop(outQueue, conn, closed)

	// close read
	// exit read loop
	// close inQueue
	// exit all handle loop
	// close outQueue
	// close write

	go func() {
		select {
		case <-s.close:
			// inQueue close when read closed and read loop exit
			// handle loop exit when inQueue closed
			log.Trace("close read as server will close")
			log.Errorn(conn.CloseRead())
		case <-closed:
			// exit go routine
		}
	}()

	// hear wait all handle complete
	wg.Wait()
	log.Trace("all handle loop exit")
	// write close when outQueue closed
	log.Trace("close outQueue")
	close(outQueue)

	<-closed
	log.Infof("close connection from %s", conn.RemoteAddr())
	log.Errorn(conn.Close())
	s.wg.Done()
}

func readLoop(inQueue chan<- *Frame, conn *net.TCPConn) (err error) {
	defer conn.CloseRead()

	size := 4 * 1024 * 1024
	conn.SetReadBuffer(size)
	buf := make([]byte, size)

	f := NewFrameHead() // an uncomplete frame
	var head, body int  // readed head, readed body
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Error(err)
			} else {
				log.Trace("read close")
			}
			log.Trace("close inQueue")
			close(inQueue)
			return err
		}

		if n == 0 { // no data
			continue
		}

		var i int
		for {
			// read head
			if head < HeadLength { // require head
				m := copy(f.head[head:], buf[i:n])
				head += m // head required length
				i += m    // data from i: buf[i:n]

				if head < HeadLength { // data not enough to head
					break
				}

				fh := f.FixedHead()
				if fh[0] != Fixed[0] || fh[1] != Fixed[1] {
					log.Error(ErrFixedHead)
					return ErrFixedHead
				}
				dl := f.DataLength()
				if dl > MaxLength {
					log.Error(ErrDataLengthExceed)
					return ErrDataLengthExceed
				}

				body = 0
				f.data = make([]byte, f.DataLength())
			}

			// read body
			m := copy(f.data[body:], buf[i:n])
			body += m // body required length
			i += m    // data from i: buf[i:n]

			if body < int(f.DataLength()) { // data not enough to body
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

func handleFrameLoop(outQueue chan<- *Frame, inQueue <-chan *Frame, h func(*Frame) *Frame, wg *sync.WaitGroup) (err error) {
	defer func() {
		if err := recover(); err != nil {
			log.Stack(err)
			go handleFrameLoop(outQueue, inQueue, h, wg)
		}
	}()
	var f *Frame
	var ok bool
	for {
		if f, ok = <-inQueue; !ok {
			log.Trace("exit hand loop")
			wg.Done()
			return nil
		}
		if f.Version() != VersionPing {
			f = h(f)
			if f == nil {
				log.Error(ErrFrameNil)
				continue
			}
		}
		outQueue <- f
	}
}

func handleLoop(outQueue chan<- *Frame, inQueue <-chan *Frame, h func([]byte) []byte, wg *sync.WaitGroup) (err error) {
	defer func() {
		if err := recover(); err != nil {
			log.Stack(err)
			go handleLoop(outQueue, inQueue, h, wg)
		}
	}()
	var f *Frame
	var ok bool
	for {
		if f, ok = <-inQueue; !ok {
			log.Trace("exit hand loop")
			wg.Done()
			return nil
		}
		if f.Version() != VersionPing {
			f.SetDataWithLength(h(f.data))
		}
		outQueue <- f
	}
}

func writeLoop(outQueue <-chan *Frame, conn *net.TCPConn, closed chan struct{}) (err error) {
	defer conn.CloseWrite()

	size := 1024 * 1024
	conn.SetWriteBuffer(size)
	buf := make([]byte, size)
	var i int
	var f *Frame
	var ok bool
	for {
		select {
		case f, ok = <-outQueue:
			if !ok {
				log.Trace("close write")
				log.Errorn(conn.CloseWrite())
				close(closed)
				return nil
			}
			if i+HeadLength+int(f.DataLength()) > size {
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
