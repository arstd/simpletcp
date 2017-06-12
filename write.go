package simpletcp

import (
	"bufio"
	"encoding/binary"
)

func (f *Frame) Write(bw *bufio.Writer) (err error) {

	// write fixed header
	if _, err = bw.Write(f.FixedHeader[:]); err != nil {
		return
	}

	// write version
	if err = bw.WriteByte(f.Version); err != nil {
		return
	}

	// write data type
	if err = bw.WriteByte(f.DataType); err != nil {
		return
	}

	// write message id
	if err = binary.Write(bw, binary.BigEndian, f.MessageId); err != nil {
		return
	}

	// write data length
	f.DataLength = uint32(len(f.Data)) // use real data length
	if err = binary.Write(bw, binary.BigEndian, f.DataLength); err != nil {
		return
	}

	// write data type
	if _, err = bw.Write(f.Reserved[:]); err != nil {
		return
	}

	// write data
	if _, err = bw.Write(f.Data); err != nil {
		return
	}

	return bw.Flush()
}
