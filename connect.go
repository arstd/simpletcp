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

	exit chan struct{}
	wg   sync.WaitGroup

	br  *bufio.Reader
	bw  *bufio.Writer
	mux sync.Mutex

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

func (c *Connect) Run() {
	go c.checkServExit()

	for {
		f, err := c.read()
		if err != nil {
			break
		}

		go func(f *Frame) {
			c.wg.Add(1)
			f = c.process(f)

			c.mux.Lock()
			c.write(f)
			c.mux.Unlock()
			c.wg.Done()
		}(f)
	}

	close(c.exit)

	c.wg.Wait()
	log.Infof("client %s leave", c.conn.RemoteAddr())
}

func (c *Connect) read() (f *Frame, err error) {
	f = NewFrame()
	_, err = io.ReadFull(c.br, f.Head)
	if err != nil {
		if err == io.EOF {
			log.Infof("client %s leave", c.conn.RemoteAddr())
		} else {
			log.Error(err)
		}
		return nil, err
	}

	log.Debug(f)

	fh := f.FixedHead()
	if fh[0] != c.serv.Fixed[0] || fh[1] != c.serv.Fixed[1] {
		log.Error(ErrFixedHead)
		return nil, ErrFixedHead
	}
	bl := f.BodyLength()
	if bl > MaxLength {
		log.Error(ErrBodyLengthExceed)
		return nil, ErrBodyLengthExceed
	}

	f.Body = make([]byte, bl)
	_, err = io.ReadFull(c.br, f.Body)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return f, nil
}

func (c *Connect) process(f *Frame) *Frame {
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
			}
		}
	}
	return f
}

func (c *Connect) write(f *Frame) {
	_, err := c.bw.Write(f.Head)
	if err != nil {
		log.Error(err)
	}

	if f.BodyLength() > 0 {
		_, err = c.bw.Write(f.Body)
		if err != nil {
			log.Error(err)
		}
	}

	err = c.bw.Flush()
	if err != nil {
		log.Error(err)
	}
}

func (c *Connect) checkServExit() {
	c.wg.Add(1)
	select {
	case <-c.serv.exit:
		log.Trace("close read as server will close")
		log.Errorn(c.conn.CloseRead())
	case <-c.exit:
	}
	c.wg.Done()
}
