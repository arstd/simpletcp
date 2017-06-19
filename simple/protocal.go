package simple

import (
	"io"
	"net"

	"github.com/arstd/log"
	"github.com/arstd/tcp"
)

type Protocol struct{}

func (p *Protocol) ReadPacket(conn *net.TCPConn) (tcp.Packet, error) {
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
	pack.Data = make([]byte, dataLength)
	if _, err := io.ReadFull(conn, pack.Data); err != nil {
		log.Error(err)
		return nil, err
	}

	return pack, nil
}
