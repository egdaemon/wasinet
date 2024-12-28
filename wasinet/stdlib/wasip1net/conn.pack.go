package wasip1net

import (
	"io"
	"net"
)

type innerpconn interface {
	io.Closer
	net.PacketConn
}
type pconn struct {
	innerpconn
}

func makePacketConn(pc innerpconn) net.PacketConn {
	return pconn{innerpconn: pc}
}
