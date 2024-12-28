package wasip1net

import (
	"net"

	"github.com/egdaemon/wasinet/wasinet/stdlib/wasip1syscall"
)

type sockaddr interface {
	Sockaddr() wasip1syscall.RawSocketAddress
}

func SetNetAddr(sotype int, dst net.Addr, src sockaddr) {
	// switch a := dst.(type) {
	// case *net.IPAddr:
	// 	a.IP, _ = sockaddrIPAndPort(src)
	// case *net.TCPAddr:
	// 	a.IP, a.Port = sockaddrIPAndPort(src)
	// case *net.UDPAddr:
	// 	a.IP, a.Port = sockaddrIPAndPort(src)
	// case *net.UnixAddr:
	// 	switch sotype {
	// 	case syscall.SOCK_STREAM:
	// 		a.Net = "unix"
	// 	case syscall.SOCK_DGRAM:
	// 		a.Net = "unixgram"
	// 	}
	// 	a.Name = sockaddrName(src)
	// default:
	// 	log.Printf("unable to set addr: %T\n", dst)
	// }
}
