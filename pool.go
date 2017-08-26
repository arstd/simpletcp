package simpletcp

import (
	"sync/atomic"

	"github.com/arstd/log"
)

const poolSize = 4096

// frame pool, head length fixed

var fp = newFramePool(poolSize)

type FramePool struct {
	c chan *Frame
}

func newFramePool(size int) *FramePool {
	fp := &FramePool{c: make(chan *Frame, size)}
	return fp
}

func (fp *FramePool) Get() *Frame {
	select {
	case f := <-fp.c:
		log.Tracef("get frame %p", f)
		return f
	default:
		log.Debug("frame pool empty")
		return newFrameHead()
	}
}

func (fp *FramePool) Put(f *Frame) {
	select {
	case fp.c <- f:
		log.Tracef("put frame %p", f)
	default:
		log.Debug("frame pool full")
	}
}

// body pool, body length is volatile

var bp = newBodyPool(poolSize)

type BodyPool struct {
	c     chan []byte
	total uint64
	count uint64

	hit  uint64
	miss uint64
}

func newBodyPool(size int) *BodyPool {
	bp := &BodyPool{
		c:     make(chan []byte, size),
		count: 1,
		total: 5,
	}
	return bp
}

func (bp *BodyPool) Get(l int) []byte {
	atomic.AddUint64(&bp.count, 1)
	atomic.AddUint64(&bp.total, uint64(l))

	select {
	case bs := <-bp.c:
		if len(bs) < l {
			miss := atomic.AddUint64(&bp.miss, 1)
			log.Debugf("miss %d: body len not enough, require %d, got %d", miss, l, len(bs))
			return make([]byte, l)
		}
		hit := atomic.AddUint64(&bp.hit, 1)
		log.Tracef("hit %d: get body %p", hit, &bs[0])
		return bs
	default:
		log.Debug("body pool empty")
		return make([]byte, l)
	}
}

func (bp *BodyPool) Put(bs []byte) {
	l := len(bs)
	avg := atomic.LoadUint64(&bp.total) / atomic.LoadUint64(&bp.count)
	if l < int(avg>>1) || l > int(avg<<3) {
		return
	}

	select {
	case bp.c <- bs:
		log.Tracef("put body %p", &bs[0])
	default:
		log.Debug("body pool full")
	}
}
