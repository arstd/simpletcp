package simple

import (
	"bufio"
	"io"

	"github.com/arstd/log"
	"github.com/arstd/tcp"
)

type Protocol struct{}

func (*Protocol) ReadPacket(conn *bufio.Reader) (tcp.Packet, error) {
	pack := NewPacketHeader()

	// read header
	if _, err := io.ReadFull(conn, pack.header); err != nil {
		if err != io.EOF {
			log.Error(err)
		}
		return nil, err
	}

	// reader data
	dataLength := pack.DataLength()
	if dataLength == 0 {
		pack.Data = nil
	} else {
		pack.Data = make([]byte, dataLength)
		if _, err := io.ReadFull(conn, pack.Data); err != nil {
			log.Error(err)
			return nil, err
		}
	}

	return pack, nil
}

func (*Protocol) WritePacket(conn *bufio.Writer, p tcp.Packet) (err error) {
	if _, err = conn.Write(p.GetHeader()); err != nil {
		return
	}
	if _, err = conn.Write(p.GetBody()); err != nil {
		return
	}
	return
}
