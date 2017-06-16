package simpletcp

import (
	"bufio"
	"fmt"
	"net"
)

type Client struct {
	Host string
	Port int

	Fixed     [2]byte // default 'Ac' (0x41 0x63)
	MaxLength uint32  // default 655356 (1<<16)

	Version  byte // default 1 (0x01)
	DataType byte // default 1 (0x01, json)

	conn net.Conn
	br   *bufio.Reader
	bw   *bufio.Writer

	nextId uint32
}

func (s *Client) NextId() uint32 {
	s.nextId++
	return s.nextId
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
	if s.Fixed != zero {
		return
	}

	if s.Fixed == [2]byte{} {
		s.Fixed = Fixed
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
			Version:   s.Version,
			DataType:  s.DataType,
			MessageId: s.NextId(),
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

	if f.DataLength > s.MaxLength {
		return nil, ErrDataLengthExceed
	}

	if err = Write(s.bw, s.Fixed, f); err != nil {
		return
	}

	return Read(s.br, s.Fixed, s.MaxLength)
}

func (s *Client) Close() {
	if s.conn != nil {
		s.conn.Close()
	}
}
