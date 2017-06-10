package simpletcp

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

func (f *Frame) Read(r io.Reader) (err error) {
	br := bufio.NewReader(r)

	// read fixed header
	if err = expectByte(br, "fixed header[0]", f.FixedHeader[0]); err != nil {
		return
	}
	if err = expectByte(br, "fixed header[1]", f.FixedHeader[1]); err != nil {
		return
	}

	// read version
	if f.Version, err = br.ReadByte(); err != nil {
		return
	}

	// read data type
	if f.DataType, err = br.ReadByte(); err != nil {
		return
	}

	// read message id
	if f.MessageId, err = readUint32(br); err != nil {
		return
	}

	// read data length
	if f.DataLength, err = readUint32(br); err != nil {
		return
	} else if f.DataLength > f.MaxLength {
		err = fmt.Errorf("data length exceed, max %d, but got %d", f.MaxLength, f.DataLength)
		return
	}

	// read reserved
	if _, err = io.ReadFull(br, f.Reserved[:]); err != nil {
		return
	}

	// read data
	return f.ReadData(br)
}

func (f *Frame) ReadData(r io.Reader) (err error) {
	if len(f.Data) < int(f.DataLength) {
		f.Data = make([]byte, f.DataLength)
	} else {
		f.Data = f.Data[:f.DataLength]
	}

	if _, err = io.ReadFull(r, f.Data); err != nil {
		return err
	}

	return
}

func expectByte(br *bufio.Reader, name string, expect byte) error {
	if b, err := br.ReadByte(); err != nil {
		return err
	} else if b != expect {
		err := fmt.Errorf("%s got %x, expect %x", name, b, expect)
		return err
	}
	return nil
}

func readUint32(br *bufio.Reader) (u uint32, err error) {
	if err := binary.Read(br, binary.BigEndian, &u); err != nil {
		return 0, err
	}
	return
}
