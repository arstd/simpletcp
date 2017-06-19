package simple

import (
	"encoding/binary"
	"errors"
)

const (
	maxDataLength = 65536
)

var (
	defaultFVT          = []byte{'A', 'c', 0x01, 0x01}
	ErrDataLengthExceed = errors.New("data length exceed")
)

type Packet struct {
	header []byte
	Data   []byte
}

func NewPacketHeader() *Packet {
	return &Packet{header: make([]byte, 16)}
}

func NewPacket(dataLength uint32) (*Packet, error) {
	if dataLength > maxDataLength {
		return nil, ErrDataLengthExceed
	}
	s := &Packet{Data: make([]byte, dataLength)}
	copy(s.header[0:4], defaultFVT)
	binary.BigEndian.PutUint32(s.header[4:8], dataLength)

	return s, nil
}

func (p *Packet) GetHeader() []byte {
	binary.BigEndian.PutUint32(p.header[8:12], uint32(len(p.Data)))
	return p.header
}

func (p *Packet) GetBody() []byte {
	return p.Data
}

func (p *Packet) FixedHeader() []byte {
	return p.header[:2]
}

func (p *Packet) Version() []byte {
	return p.header[2:3]
}

func (p *Packet) DateType() []byte {
	return p.header[3:4]
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
