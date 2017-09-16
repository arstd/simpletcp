package simpletcp

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	ErrFixedHead        = errors.New("fixed Head not 0x41 0x63")
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

const MaxLength uint32 = 1<<16 - HeadLength

var Reserved [4]byte = [4]byte{0, 0, 0, 0}

type Frame struct {
	Head []byte
	Body []byte
}

func NewFrame() *Frame {
	return &Frame{Head: make([]byte, HeadLength)}
}

func NewFrameDefault() *Frame {
	f := NewFrame()
	f.Head[0] = 'A'
	f.Head[1] = 'c'
	f.Head[2] = Version1
	f.Head[3] = BodyTypeJSON
	return f
}

func (f *Frame) SetVersion(version byte) {
	f.Head[2] = version
}

func (f *Frame) SetMessageId(messageId uint32) {
	binary.BigEndian.PutUint32(f.Head[4:8], messageId)
}

func (f *Frame) SetReserved(reserved uint32) {
	binary.BigEndian.PutUint32(f.Head[12:16], reserved)
}

func (f *Frame) SetBodyOnly(body []byte) {
	f.Body = body
}

func (f *Frame) SetBodyWithLength(body []byte) {
	binary.BigEndian.PutUint32(f.Head[8:12], uint32(len(body)))
	f.SetBodyOnly(body)
}

func (f *Frame) SetBodyLength(bl uint32) {
	binary.BigEndian.PutUint32(f.Head[8:12], bl)
}

func (f *Frame) FixedHead() []byte {
	return f.Head[0:2]
}

func (f *Frame) Version() byte {
	return f.Head[2]
}

func (f *Frame) BodyType() byte {
	return f.Head[3]
}

func (f *Frame) MessageId() uint32 {
	return binary.BigEndian.Uint32(f.Head[4:8])
}

func (f *Frame) BodyLength() uint32 {
	return binary.BigEndian.Uint32(f.Head[8:12])
}

func (f *Frame) Reserved() []byte {
	return f.Head[12:16]
}

func (f *Frame) String() string {
	return fmt.Sprintf("%c %x %x %d %d %x: %s", f.FixedHead(), f.Version(),
		f.BodyType(), f.MessageId(), f.BodyLength(), f.Reserved(), f.Body)
}
