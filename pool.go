package simpletcp

import "sync"

// frame pool, head length fixed

var framePool = sync.Pool{
	New: func() interface{} {
		return &Frame{head: make([]byte, 16)}
	},
}

func getFrame() *Frame {
	return framePool.Get().(*Frame)
}

func putFrame(f *Frame) {
	f.data = nil
	framePool.Put(f)
}

// body pool, body length is volatile
var bodyPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 256)
	},
}

func getBody(l int) []byte {
	bs := bodyPool.Get().([]byte)
	if len(bs) < l {
		bs = make([]byte, l)
	}
	return bs
}

func putBody(bs []byte) {
	bodyPool.Put(bs)
}
