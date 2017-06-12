package simpletcp

import (
	"bufio"
	"fmt"
	"net"
)

type Client struct {
	Host string
	Port int

	FixedHeader [2]byte // default 'Ac' (0x41 0x63)
	Version     byte    // default 1 (0x01)
	DataType    byte    // default 1 (0x01, json)
	MaxLength   uint32  // default 655356 (1<<16)

	conn net.Conn
	br   *bufio.Reader
	bw   *bufio.Writer

	messageId uint32
}

func (s *Client) NextMessageId() uint32 {
	s.messageId++
	return s.messageId
}

func (s *Client) connect() error {
	if s.conn != nil {
		return nil
	}

	var err error
	s.conn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", s.Host, s.Port))
	if err != nil {
		return err
	}

	s.br = bufio.NewReader(s.conn)
	s.bw = bufio.NewWriter(s.conn)

	return nil
}

var zero [2]byte

func (s *Client) check() {
	if s.FixedHeader != zero {
		return
	}

	if s.FixedHeader == [2]byte{} {
		s.FixedHeader = FixedHeader
	}
	if s.Version == 0 {
		s.Version = Version1
	}
	if s.DataType == 0 {
		s.DataType = DataTypeJSON
	}
	if s.MaxLength == 0 {
		s.MaxLength = MaxLength
	}
}

func (s *Client) Send(data []byte) ([]byte, error) {
	s.check()

	f := &Frame{
		Header: Header{
			FixedHeader: s.FixedHeader,
			Version:     s.Version,
			DataType:    s.DataType,
			MaxLength:   s.MaxLength,
		},
		Data: data,
	}

	var received *Frame
	var err error
	if received, err = s.SendFrame(f); err != nil {
		return nil, err
	}

	return received.Data, nil
}

func (s *Client) SendFrame(f *Frame) (received *Frame, err error) {
	if f == nil {
		return nil, ErrFrameNil
	}

	s.check()

	if err = s.connect(); err != nil {
		return
	}

	if f.MessageId == 0 {
		f.MessageId = s.NextMessageId()
	}
	if f.DataLength > s.MaxLength {
		return nil, ErrDataLengthExceed
	}

	if err = Write(s.bw, f); err != nil {
		return
	}

	return Read(s.br, s.FixedHeader, s.MaxLength)
}

func (s *Client) Close() {
	if s.conn != nil {
		s.conn.Close()
	}
}
