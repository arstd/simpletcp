package simpletcp

import "github.com/arstd/log"

const poolSize = 4096

// frame pool, head length fixed

var fp = newFramePool(poolSize)

type FramePool struct {
	c chan *Frame
}

func newFramePool(size int) *FramePool {
	fp := &FramePool{c: make(chan *Frame, size)}
	for i := 0; i < size; i++ {
		fp.c <- newFrameHead()
	}
	return fp
}

func (fp *FramePool) Get() *Frame {
	select {
	case f := <-fp.c:
		return f
	default:
		log.Info("frame pool empty")
		return newFrameHead()
	}
}

func (fp *FramePool) Put(f *Frame) {
	select {
	case fp.c <- f:
	default:
		log.Info("frame pool full")
	}
}

// body pool, body length is volatile

var bp = newBodyPool(poolSize)

const maxBytes = 4096

type BodyPool struct {
	c chan []byte
}

func newBodyPool(size int) *BodyPool {
	bp := &BodyPool{c: make(chan []byte, size)}
	for i := 0; i < size; i++ {
		bp.c <- make([]byte, maxBytes/10)
	}
	return bp
}

func (bp *BodyPool) Get(l int) []byte {
	select {
	case bs := <-bp.c:
		if len(bs) < l {
			return make([]byte, l)
		}
		return bs
	default:
		log.Info("body pool empty")
		return make([]byte, l)
	}
}

func (bp *BodyPool) Put(bs []byte) {
	select {
	case bp.c <- bs:
	default:
		log.Info("body pool full")
	}
}
