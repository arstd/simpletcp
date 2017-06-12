package simpletcp

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"sync"
)

type Client struct {
	sync.Mutex

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

	var received Frame
	var err error
	if err = s.SendFrame(f, &received); err != nil {
		return nil, err
	}

	return received.Data, nil
}

var ErrFrameNil = errors.New("frame is nil")

func (s *Client) SendFrame(f *Frame, received *Frame) (err error) {
	if f == nil || received == nil {
		return ErrFrameNil
	}

	s.check()

	s.Lock()
	defer s.Unlock()

	if err = s.connect(); err != nil {
		return err
	}

	if f.MessageId == 0 {
		f.MessageId = s.NextMessageId()
	}

	if err = f.Write(s.bw); err != nil {
		return err
	}

	if received.FixedHeader == zero {
		received.FixedHeader = s.FixedHeader
	}
	if received.MaxLength == 0 {
		received.MaxLength = s.MaxLength
	}
	if err = received.Read(s.br); err != nil {
		return err
	}

	return nil
}

func (s *Client) Close() {
	if s.conn != nil {
		s.conn.Close()
	}
}
