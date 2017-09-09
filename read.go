package simpletcp

import (
	"io"
	"net"
	"time"

	"github.com/arstd/log"
)

func (c *Connect) readLoop() (err error) {
	defer c.conn.CloseRead()

	buf := make([]byte, startBufSize)

	f := NewFrame()            // an uncomplete frame
	var head, body int         // readed head, readed body
	timeout := 5 * time.Minute // close conn if 5m no data
	for {
		c.conn.SetReadDeadline(time.Now().Add(timeout))
		n, err := c.conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Error(err)
			}
			if oe, ok := err.(net.Error); ok && oe.Timeout() {
				log.Infof("%s read timeout: %s", c.conn.RemoteAddr(), oe)
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
				dl := f.BodyLength()
				if dl > MaxLength {
					log.Error(ErrBodyLengthExceed)
					return ErrBodyLengthExceed
				}

				body = 0
				f.NewBody(int(dl))
			}

			// read body
			m := copy(f.Body[body:], buf[i:n])
			body += m // body required length
			i += m    // data from i: buf[i:n]

			if body < int(f.BodyLength()) { // data not enough to body
				break
			}
			// frame complete

			select {
			case c.inQueue <- f:
			default:
				log.Warn("inQueue full")
				c.inQueue <- f
			}

			// another frame
			head = 0
			f = NewFrame()

			if i >= n {
				break
			}
		}

		// grow: if read full buffer and buffer length less than max buffer size
		// shrink: if read half buffer and buffer length more than min buffer size
		if n == len(buf) && len(buf) < maxBufSize {
			grow := len(buf) * 2
			if grow > maxBufSize {
				grow = maxBufSize
			}
			log.Infof("read buffer grow from %d to %d", len(buf), grow)
			buf = make([]byte, grow)
		} else if n < len(buf)/2 && len(buf) > minBufSize {
			shrink := len(buf) / 2
			if shrink < minBufSize {
				shrink = minBufSize
			}
			log.Infof("read buffer shrink from %d to %d", len(buf), shrink)
			buf = make([]byte, shrink)
		}
	}
}
