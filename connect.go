package simpletcp

import (
	"net"
	"sync"

	"github.com/arstd/log"
)

// For controlling dynamic buffer sizes.
const (
	startBufSize = 512
	minBufSize   = 128
	maxBufSize   = 65536
)

type Connect struct {
	conn   *net.TCPConn
	wg     sync.WaitGroup
	closed chan struct{} // can read after connect closed

	queueSize         int32
	inQueue, outQueue chan *Frame

	handle      func([]byte) []byte // one of handlers must not nil
	handleFrame func(*Frame) *Frame
}

func NewConnect(conn *net.TCPConn, queueSize int32, handle func([]byte) []byte, handleFrame func(*Frame) *Frame) *Connect {
	conn.SetNoDelay(true)

	return &Connect{
		conn:   conn,
		closed: make(chan struct{}),

		queueSize: queueSize,
		inQueue:   make(chan *Frame, queueSize),
		outQueue:  make(chan *Frame, queueSize),

		handle:      handle,
		handleFrame: handleFrame,
	}
}

func (c *Connect) Process(procNum int, closeSignal <-chan struct{}) {
	go c.waitSignal(closeSignal)

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
