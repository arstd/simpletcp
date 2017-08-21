package simpletcp

import "github.com/arstd/log"

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
		if f.Version() != VersionPing {
			f = c.handleFrame(f)
			if f == nil {
				log.Error(ErrFrameNil)
				continue
			}
		}

		select {
		case c.outQueue <- f:
		default:
			log.Warn("outQueue full")
			c.outQueue <- f
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
		if f.Version() != VersionPing {
			f.SetBodyWithLength(c.handle(f.Body))
		}

		select {
		case c.outQueue <- f:
		default:
			log.Warn("outQueue full")
			c.outQueue <- f
		}
	}
}
