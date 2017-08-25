package simpletcp

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/arstd/log"
)

var (
	ErrFixedHead        = errors.New("fixed head not 0x41 0x63")
	ErrFrameNil         = errors.New("frame is nil")
	ErrBodyLengthExceed = errors.New("data length exceed")
)

const (
	HeadLength       = 16
	VersionPing byte = 0x00
	Version1    byte = 0x01
)

var Fixed = [2]byte{'A', 'c'}

const (
	BodyTypeJSON     byte = 0x01
	BodyTypeProtobuf byte = 0x02
	BodyTypeXML      byte = 0x03
	BodyTypePlain    byte = 0x04
)

const MaxLength uint32 = 1 << 16

var Reserved [4]byte = [4]byte{0, 0, 0, 0}

type Frame struct {
	head             []byte
	Body, underlying []byte
}

func newFrameHead() *Frame {
	return &Frame{head: make([]byte, HeadLength)}
}

// NewFrameHead return frame with blank head
func NewFrame() *Frame {
	return fp.Get()
}

func (f *Frame) NewBody(l int) {
	f.underlying = bp.Get(l)
	f.Body = f.underlying[:l]
}

func (f *Frame) Recycle() {
	f.RecycleBody()
	fp.Put(f)
}

func (f *Frame) RecycleBody() {
	if f.underlying != nil {
		bp.Put(f.underlying)
	}
	f.underlying = nil
	f.Body = nil
}

func NewFrameDefault() *Frame {
	f := NewFrame()
	f.head[0] = 'A'
	f.head[1] = 'c'
	f.head[2] = Version1
	f.head[3] = BodyTypeJSON
	return f
}

func (f *Frame) SetVersion(version byte) {
	f.head[2] = version
}

func (f *Frame) SetMessageId(messageId uint32) {
	binary.BigEndian.PutUint32(f.head[4:8], messageId)
}

func (f *Frame) SetReserved(reserved uint32) {
	binary.BigEndian.PutUint32(f.head[12:16], reserved)
}

func (f *Frame) SetBody(body []byte) {
	f.Body = body
	if len(body) == cap(body) {
		log.Tracef("slice (%p len=%d, cap=%d)", &body[0], len(body), cap(body))
		f.underlying = body
	}
	// unsafe code, panic:
	// fatal error: sweep increased allocation count

	// if len(body) < cap(body) {
	// 	uh := (*reflect.SliceHeader)(unsafe.Pointer(&f.underlying))
	// 	uh.Data = uintptr(unsafe.Pointer(&body))
	// 	uh.Cap = cap(body)
	// 	uh.Len = cap(body)
	// 	log.Tracef("slice underlying %p(len=%d, cap=%d) from slice %p(len=%d, cap=%d)",
	// 		&f.underlying[0], len(f.underlying), cap(f.underlying), &body[0], len(body), cap(body))
	// }
}

func (f *Frame) SetBodyWithLength(body []byte) {
	binary.BigEndian.PutUint32(f.head[8:12], uint32(len(body)))
	f.SetBody(body)
}

func (f *Frame) Head() []byte {
	return f.head
}

func (f *Frame) FixedHead() []byte {
	return f.head[0:2]
}

func (f *Frame) Version() byte {
	return f.head[2]
}

func (f *Frame) BodyType() byte {
	return f.head[3]
}

func (f *Frame) MessageId() uint32 {
	return binary.BigEndian.Uint32(f.head[4:8])
}

func (f *Frame) BodyLength() uint32 {
	return binary.BigEndian.Uint32(f.head[8:12])
}

func (f *Frame) Reserved() []byte {
	return f.head[12:16]
}

func (f *Frame) String() string {
	return fmt.Sprintf("%c %x %x %d %d %x: %s", f.FixedHead(), f.Version(),
		f.BodyType(), f.MessageId(), f.BodyLength(), f.Reserved(), f.Body)
}
