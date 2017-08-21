package simpletcp

import "github.com/arstd/log"

func (c *Connect) writeLoop() (err error) {
	defer c.conn.CloseWrite()

	buf := make([]byte, c.writeBufferSize)
	var i int
	var f *Frame
	var ok bool
	for {
		select {
		case f, ok = <-c.outQueue:

		default:
			if i > 0 { // buf not full but no frame
				if _, err := c.conn.Write(buf[:i]); err != nil {
					log.Error(err)
					return err
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
		if i+HeadLength+int(f.BodyLength()) > c.writeBufferSize {
			if _, err := c.conn.Write(buf[:i]); err != nil {
				log.Error(err)
				return err
			}
			i = 0
		}
		i += copy(buf[i:], f.head)
		i += copy(buf[i:], f.Body)
		f.Recycle()
	}
}
