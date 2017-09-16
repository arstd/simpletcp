package simpletcp

import (
	"bufio"
	"io"
	"net"
	"sync"

	"github.com/arstd/log"
)

type Connect struct {
	conn *net.TCPConn
	wg   sync.WaitGroup
	exit chan struct{} // can read after connect exit

	br *bufio.Reader
	bw *bufio.Writer

	serv *Server
}

func NewConnect(conn *net.TCPConn, serv *Server) *Connect {
	return &Connect{
		conn: conn,
		exit: make(chan struct{}),

		br: bufio.NewReader(conn),
		bw: bufio.NewWriter(conn),

		serv: serv,
	}
}

func (c *Connect) Process(closeSignal <-chan struct{}) {
	go c.waitSignal(closeSignal)

	for {
		f := NewFrame()
		_, err := io.ReadFull(c.br, f.Head)
		if err != nil {
			if err == io.EOF {
				log.Infof("client %s leave", c.conn.RemoteAddr())
			} else {
				log.Error(err)
			}
			c.conn.CloseRead()
			close(c.exit)
			break
		}

		log.Debug(f)

		fh := f.FixedHead()
		if fh[0] != c.serv.Fixed[0] || fh[1] != c.serv.Fixed[1] {
			log.Error(ErrFixedHead)
			c.conn.CloseRead()
			break
		}
		bl := f.BodyLength()
		if bl > MaxLength {
			log.Error(ErrBodyLengthExceed)
			c.conn.CloseRead()
			break
		}

		f.Body = make([]byte, bl)
		_, err = io.ReadFull(c.br, f.Body)
		if err != nil {
			log.Error(err)
			c.conn.CloseRead()
			break
		}

		log.Debug(f)

		go c.handle(f)
	}

	c.wg.Wait()
	<-c.exit

	log.Warn("close process")
}

func (c *Connect) handle(f *Frame) {
	if f.Version() != VersionPing {
		if c.serv.Handle != nil {
			out := c.serv.Handle(f.Body)
			f.SetBodyWithLength(out)
		} else if c.serv.HandleFrame != nil {
			f = c.serv.HandleFrame(f)
			bl := uint32(len(f.Body))
			if bl < MaxLength {
				f.SetBodyLength(bl)
			} else {
				log.Error(ErrBodyLengthExceed)
				f.SetBodyWithLength(nil)
				return
			}
		} else {
			close(c.exit)
			log.Fatal("one of Handle or HandleFrame must be not nil")
		}
	}

	c.write(f)
}

func (c *Connect) write(f *Frame) {
	_, err := c.bw.Write(f.Head)
	if err != nil {
		log.Error(err)
		c.conn.CloseWrite()
		close(c.exit)
	}
	_, err = c.bw.Write(f.Body)
	if err != nil {
		log.Error(err)
		c.conn.CloseWrite()
		close(c.exit)
	}

	err = c.bw.Flush()
	if err != nil {
		log.Error(err)
		c.conn.CloseWrite()
		close(c.exit)
	}
}

func (c *Connect) waitSignal(closeSignal <-chan struct{}) {
	c.wg.Add(1)
	select {
	case <-closeSignal:
		log.Trace("close read as server will close")
		log.Errorn(c.conn.CloseRead())
		close(c.exit)
	case <-c.exit: // connect exit, must exit this method
		// exit go routine
	}
	c.wg.Done()

	log.Warn("close process")
}
