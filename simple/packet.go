package simple

import (
	"encoding/binary"
	"errors"
)

const (
	maxDataLength = 65536
	DataTypePing  = 0x00
)

var (
	defaultFVT          = []byte{'A', 'c', 0x01, 0x01}
	ErrDataLengthExceed = errors.New("data length exceed")
)

type Packet struct {
	header []byte
	Data   []byte
}

func NewPacket(fixed [2]byte, version, dataType byte, messageId uint32, reserved [4]byte) *Packet {
	p := &Packet{header: make([]byte, 16)}
	p.header[0] = fixed[0]
	p.header[1] = fixed[1]
	p.header[2] = version
	p.header[3] = dataType
	binary.BigEndian.PutUint32(p.header[4:8], messageId)
	copy(p.header[12:16], reserved[:])
	return p
}

func NewPacketMessageId(messageId uint32) *Packet {
	p := &Packet{header: make([]byte, 16)}
	copy(p.header[0:4], defaultFVT)
	binary.BigEndian.PutUint32(p.header[4:8], messageId)
	return p
}

func NewPacketHeader() *Packet {
	return &Packet{header: make([]byte, 16)}
}

func (p *Packet) SetMessageId(messageId uint32) {
	binary.BigEndian.PutUint32(p.header[4:8], messageId)
}

func (p *Packet) SetReserved(reserved uint32) {
	binary.BigEndian.PutUint32(p.header[12:16], reserved)
}

func (p *Packet) GetHeader() []byte {
	binary.BigEndian.PutUint32(p.header[8:12], uint32(len(p.Data)))
	return p.header
}

func (p *Packet) GetBody() []byte {
	return p.Data
}

func (p *Packet) FixedHeader() []byte {
	return p.header[0:2]
}

func (p *Packet) Version() byte {
	return p.header[2]
}

func (p *Packet) DateType() byte {
	return p.header[3]
}

func (p *Packet) MessageId() uint32 {
	return binary.BigEndian.Uint32(p.header[4:8])
}

func (p *Packet) DataLength() uint32 {
	return binary.BigEndian.Uint32(p.header[8:12])
}

func (p *Packet) Reserved() []byte {
	return p.header[12:16]
}
