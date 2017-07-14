package simpletcp

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	ErrFixedHead        = errors.New("fixed head not 0x41 0x63")
	ErrFrameNil         = errors.New("frame is nil")
	ErrDataLengthExceed = errors.New("data length exceed")
)

const (
	HeadLength       = 16
	VersionPing byte = 0x00
	Version1    byte = 0x01
)

var Fixed = [2]byte{'A', 'c'}

const (
	DataTypeJSON     byte = 0x01
	DataTypeProtobuf byte = 0x02
	DataTypeXML      byte = 0x03
	DataTypePlain    byte = 0x04
)

const MaxLength uint32 = 1 << 16

var Reserved [4]byte = [4]byte{0, 0, 0, 0}

type Frame struct {
	head []byte
	data []byte
}

// NewFrameHead return frame with blank head
func NewFrameHead() *Frame {
	return &Frame{head: make([]byte, 16)}
}

func NewFrameDefault() *Frame {
	f := NewFrameHead()
	f.head[0] = 'A'
	f.head[1] = 'c'
	f.head[2] = Version1
	f.head[3] = DataTypeJSON
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

func (f *Frame) SetData(data []byte) {
	f.data = data
}

func (f *Frame) SetDataWithLength(data []byte) {
	binary.BigEndian.PutUint32(f.head[8:12], uint32(len(data)))
	f.data = data
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

func (f *Frame) DataType() byte {
	return f.head[3]
}

func (f *Frame) MessageId() uint32 {
	return binary.BigEndian.Uint32(f.head[4:8])
}

func (f *Frame) DataLength() uint32 {
	return binary.BigEndian.Uint32(f.head[8:12])
}

func (f *Frame) Reserved() []byte {
	return f.head[12:16]
}

func (f *Frame) Data() []byte {
	return f.data
}

func (f *Frame) String() string {
	return fmt.Sprintf("%c %x %x %d %d %x: %s", f.FixedHead(), f.Version(),
		f.DataType(), f.MessageId(), f.DataLength(), f.Reserved(), f.Data())
}
