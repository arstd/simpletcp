package simpletcp

import "errors"

var Fixed = [2]byte{'A', 'c'}

const (
	VersionPing byte = 0x00
	Version1    byte = 0x01
)

const (
	DataTypeJSON     byte = 0x01
	DataTypeProtobuf byte = 0x02
	DataTypeXML      byte = 0x03
	DataTypePlain    byte = 0x04
)

const MaxLength uint32 = 1 << 16

var Reserved [4]byte

type Header struct {
	// FixedHeader [2]byte
	Version  byte
	DataType byte

	MessageId  uint32
	DataLength uint32

	Reserved [4]byte
}

type Frame struct {
	Header
	Data []byte
}

var (
	ErrFrameNil         = errors.New("frame is nil")
	ErrDataLengthExceed = errors.New("data length exceed")
)
