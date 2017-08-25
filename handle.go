package simpletcp

import (
	"unsafe"

	"github.com/arstd/log"
)

func (c *Connect) handleFrameLoop() (err error) {
	defer func() {
		if err := recover(); err != nil {
			log.Stack(err)
			go c.handleFrameLoop()
		}
	}()
	var in, out *Frame
	var ok bool
	for {
		if in, ok = <-c.inQueue; !ok {
			log.Trace("exit hand loop")
			c.wg.Done()
			return nil
		}
		if in.Version() != VersionPing {
			out = c.handleFrame(in)
			if out == nil {
				log.Error(ErrFrameNil)
				continue
			}
			if unsafe.Pointer(&out.Body) != unsafe.Pointer(&in.Body) {
				if unsafe.Pointer(out) != unsafe.Pointer(in) {
					in.Recycle()
				} else {
					in.RecycleBody()
				}
			}
		}

		select {
		case c.outQueue <- out:
		default:
			log.Warn("outQueue full")
			c.outQueue <- out
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
	var out []byte
	var ok bool
	for {
		if f, ok = <-c.inQueue; !ok {
			log.Trace("exit hand loop")
			c.wg.Done()
			return nil
		}
		if f.Version() != VersionPing {
			out = c.handle(f.Body)
			if unsafe.Pointer(&out) != unsafe.Pointer(&f.Body) {
				f.RecycleBody()
			}
			f.SetBodyWithLength(out)
		}

		select {
		case c.outQueue <- f:
		default:
			log.Warn("outQueue full")
			c.outQueue <- f
		}
	}
}
