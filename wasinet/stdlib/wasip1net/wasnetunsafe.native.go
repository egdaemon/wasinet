//go:build !wasip1

package wasip1net

import (
	"net"
	"os"
)

func newFile(fd int, name string) *os.File {
	return os.NewFile(uintptr(fd), name)
}

func newConn(f *os.File) (c net.Conn, err error) {
	return net.FileConn(f)
}

func PacketConn(fd uintptr, family int, sotype int, netw string, laddr, raddr net.Addr) (net.PacketConn, error) {
	pc, err := net.FilePacketConn(Socket(uintptr(fd)))
	if err != nil {
		return nil, err
	}
	return makePacketConn(pc), nil
}
