package simpletcp

import (
	"github.com/arstd/log"
)

func (c *Connect) writeLoop() (err error) {
	defer c.conn.CloseWrite()

	if maxBufSize < MaxLength+HeadLength {
		panic("max buffer size must larger than max of frame data length")
	}

	buf := make([]byte, startBufSize)
	var i int
	var f *Frame
	var ok bool
	for {
		select {
		case f, ok = <-c.outQueue:
		default:
			if i > 0 { // buf not full but no frame
				log.Tracef("> %d", i)
				if _, err := c.conn.Write(buf[:i]); err != nil {
					log.Error(err)
					return err
				}

				// shrink: if read half buffer and buffer length more than min buffer size
				if i < len(buf)/2 && len(buf) > minBufSize {
					shrink := len(buf) / 2
					if shrink < minBufSize {
						shrink = minBufSize
					}
					log.Infof("write buffer shrink from %d to %d", len(buf), shrink)
					buf = make([]byte, shrink)
				}

				i = 0
			}

			f, ok = <-c.outQueue
		}
		if !ok {
			log.Trace("close write")
			log.Errorn(c.conn.CloseWrite())
			close(c.closed)
			return nil
		}
		log.Trace(f)

		frameLen := HeadLength + int(f.BodyLength())
		if i+frameLen > len(buf) {
			if i > 0 {
				if _, err := c.conn.Write(buf[:i]); err != nil {
					log.Error(err)
					return err
				}
				i = 0
			}
			// grow: ifbuffer full and buffer length less than max buffer size
			if len(buf) < maxBufSize {
				grow := len(buf) * 2
				if grow < frameLen {
					grow = frameLen
				}
				if grow > maxBufSize {
					grow = maxBufSize
				}
				log.Infof("write buffer grow from %d to %d", len(buf), grow)
				buf = make([]byte, grow)
			}
		}
		i += copy(buf[i:], f.head)
		i += copy(buf[i:], f.Body)
		f.Recycle()
	}
}
