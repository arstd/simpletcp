package simple

import (
	"github.com/arstd/log"
	"github.com/arstd/tcp"
)

type Callback func([]byte) []byte

func (Callback) OnConnect(c *tcp.Conn) bool {
	addr := c.GetRawConn().RemoteAddr()
	log.Infof("connect from %s", addr)
	return true
}

func (cb Callback) OnMessage(c *tcp.Conn, p tcp.Packet) (tcp.Packet, bool) {
	pack := p.(*Packet)

	res := cb(pack.Data)

	pack.Data = res
	return pack, true
}

func (Callback) OnClose(c *tcp.Conn) {
	addr := c.GetRawConn().RemoteAddr()
	log.Infof("close connect of %s", addr)
}
