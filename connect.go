package simpletcp

import (
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/arstd/log"
)

type Connect struct {
	conn   *net.TCPConn
	wg     sync.WaitGroup
	closed chan struct{} // can read after connect closed

	readBufferSize, writeBufferSize int

	queueSize         int32
	inQueue, outQueue chan *Frame
	inPlus, inMinus   int32
	outPlus, outMinus int32

	handle      func([]byte) []byte // one of handlers must not nil
	handleFrame func(*Frame) *Frame
}

func NewConnect(conn *net.TCPConn, queueSize int32, readBufferSize, writeBufferSize int, handle func([]byte) []byte, handleFrame func(*Frame) *Frame) *Connect {
	conn.SetNoDelay(true)

	return &Connect{
		conn:   conn,
		closed: make(chan struct{}),

		readBufferSize:  readBufferSize,
		writeBufferSize: writeBufferSize,

		queueSize: queueSize,
		inQueue:   make(chan *Frame, queueSize),
		outQueue:  make(chan *Frame, queueSize),

		handle:      handle,
		handleFrame: handleFrame,
	}
}

func (c *Connect) Process(procNum int, closeSignal <-chan struct{}) {
	go c.waitSignal(closeSignal)

	go c.printQueueLen()

	go c.readLoop()

	for i := 0; i < procNum; i++ {
		c.wg.Add(1)
		if c.handleFrame != nil {
			go c.handleFrameLoop()
		} else {
			go c.handleLoop()
		}
	}

	go c.writeLoop()

	// close read
	// exit read loop
	// close inQueue
	// exit all handle loop
	// close outQueue
	// close write

	// hear wait all handle complete, frame sent to outQueue
	c.wg.Wait()
	log.Trace("all handle loop exit")
	// close outQueue, write will close when outQueue closed
	log.Trace("close outQueue")
	close(c.outQueue)

	// wait all data write complete
	<-c.closed
	log.Infof("close connection from %s", c.conn.RemoteAddr())
	log.Errorn(c.conn.Close())
}

func (c *Connect) readLoop() (err error) {
	defer c.conn.CloseRead()

	buf := make([]byte, c.readBufferSize)

	f := NewFrameHead() // an uncomplete frame
	var head, body int  // readed head, readed body
	for {
		n, err := c.conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Error(err)
			} else {
				log.Trace("read close")
			}
			log.Trace("close inQueue")
			close(c.inQueue)
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

			select {
			case c.inQueue <- f:
				atomic.AddInt32(&c.inPlus, 1)
			default:
				log.Warn("inQueue full")
				c.inQueue <- f
				atomic.AddInt32(&c.inPlus, 1)
			}

			// another frame
			head = 0
			f = NewFrameHead()

			if i >= n {
				break
			}
		}
	}
}

func (c *Connect) handleFrameLoop() (err error) {
	defer func() {
		if err := recover(); err != nil {
			log.Stack(err)
			go c.handleFrameLoop()
		}
	}()
	var f *Frame
	var ok bool
	for {
		if f, ok = <-c.inQueue; !ok {
			log.Trace("exit hand loop")
			c.wg.Done()
			return nil
		}
		atomic.AddInt32(&c.inMinus, 1)
		if f.Version() != VersionPing {
			f = c.handleFrame(f)
			if f == nil {
				log.Error(ErrFrameNil)
				continue
			}
		}

		select {
		case c.outQueue <- f:
			atomic.AddInt32(&c.outPlus, 1)
		default:
			log.Warn("outQueue full")
			c.outQueue <- f
			atomic.AddInt32(&c.outPlus, 1)
		}
	}
}

func (c *Connect) handleLoop() (err error) {
	defer func() {
		if err := recover(); err != nil {
			log.Stack(err)
			go c.handleLoop()
		}
	}()
	var f *Frame
	var ok bool
	for {
		if f, ok = <-c.inQueue; !ok {
			log.Trace("exit hand loop")
			c.wg.Done()
			return nil
		}
		atomic.AddInt32(&c.inMinus, 1)
		if f.Version() != VersionPing {
			f.SetDataWithLength(c.handle(f.data))
		}

		select {
		case c.outQueue <- f:
			atomic.AddInt32(&c.outPlus, 1)
		default:
			log.Warn("outQueue full")
			c.outQueue <- f
			atomic.AddInt32(&c.outPlus, 1)
		}
	}
}

func (c *Connect) writeLoop() (err error) {
	defer c.conn.CloseWrite()

	buf := make([]byte, c.writeBufferSize)
	var i int
	var f *Frame
	var ok bool
	timeout := 100 * time.Microsecond
	timer := time.NewTimer(timeout)
	for {
		select {
		case f, ok = <-c.outQueue:

		default:

			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(timeout)

			select {
			case f, ok = <-c.outQueue:

			case <-timer.C:

				if i > 0 { // buf not full but no frame
					if _, err := c.conn.Write(buf[:i]); err != nil {
						log.Error(err)
						return err
					}
					i = 0
				}
				continue // goto for next iterator
			}
		}

		if !ok {
			log.Trace("close write")
			log.Errorn(c.conn.CloseWrite())
			close(c.closed)
			return nil
		}
		atomic.AddInt32(&c.outMinus, 1)
		if i+HeadLength+int(f.DataLength()) > c.writeBufferSize {
			if _, err := c.conn.Write(buf[:i]); err != nil {
				log.Error(err)
				return err
			}
			i = 0
		}
		i += copy(buf[i:], f.head)
		i += copy(buf[i:], f.data)
	}
}

func (c *Connect) waitSignal(closeSignal <-chan struct{}) {
	select {
	case <-closeSignal: // server will close, must close this connect
		// inQueue close when read closed and read loop exit
		// handle loop exit when inQueue closed
		log.Trace("close read as server will close")
		log.Errorn(c.conn.CloseRead())
	case <-c.closed: // connect closed, must exit this method
		// exit go routine
	}
}

func (c *Connect) printQueueLen() {
	timer := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-c.closed:
			timer.Stop()
			return
		case <-timer.C:
			inPlus := atomic.SwapInt32(&c.inPlus, 0)
			inMinus := atomic.SwapInt32(&c.inMinus, 0)
			outPlus := atomic.SwapInt32(&c.outPlus, 0)
			outMinus := atomic.SwapInt32(&c.outMinus, 0)

			if inPlus == 0 && outPlus == 0 &&
				inMinus == 0 && outMinus == 0 {
				continue
			}

			in, out := inPlus-inMinus, outPlus-outMinus

			p := log.Infof
			if in == c.queueSize || out == c.queueSize {
				p = log.Errorf
			} else if in*5 > c.queueSize*4 || out*5 > c.queueSize*4 {
				p = log.Warnf
			}
			p("connect: %s,\tin: %d - %d = %d,\tout: %d - %d = %d",
				c.conn.RemoteAddr(), inPlus, inMinus, in, outPlus, outMinus, out)
		}
	}
}
