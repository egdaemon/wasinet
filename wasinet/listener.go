package wasinet

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"syscall"

	"github.com/egdaemon/wasinet/wasinet/stdlib/wasip1net"
	"github.com/egdaemon/wasinet/wasinet/stdlib/wasip1syscall"
)

// Listen announces on the local network address.
func Listen(ctx context.Context, network, address string) (net.Listener, error) {
	switch network {
	case "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6", "unix":
	default:
		return nil, unsupportedNetwork(network, address)
	}
	addrs, err := wasip1syscall.LookupAddress(ctx, oplisten, network, address)
	if err != nil {
		return nil, netOpErr(oplisten, unresolvedaddr(network, address), err)
	}

	firstaddr := addrs[0]
	lstn, err := listenAddr(firstaddr)
	return lstn, netOpErr(oplisten, firstaddr, err)
}

// ListenPacket creates a listening packet connection.
func ListenPacket(ctx context.Context, network, address string) (net.PacketConn, error) {
	switch network {
	case "udp", "udp4", "udp6", "unixgram":
	default:
		return nil, unsupportedNetwork(network, address)
	}
	addrs, err := wasip1syscall.LookupAddress(ctx, oplisten, network, address)
	if err != nil {
		return nil, netOpErr(oplisten, unresolvedaddr(network, address), err)
	}
	conn, err := listenPacketAddr(addrs[0])
	return conn, netOpErr(oplisten, addrs[0], err)
}

func unsupportedNetwork(network, address string) error {
	return fmt.Errorf("unsupported network: %s://%s", network, address)
}

func listenAddr(addr net.Addr) (net.Listener, error) {
	fd, err := wasip1syscall.Socket(wasip1syscall.NetaddrAFFamily(addr), syscall.SOCK_STREAM, netaddrproto(addr))
	if err != nil {
		return nil, os.NewSyscallError("socket", err)
	}
	defer func() {
		if fd >= 0 {
			syscall.Close(fd)
		}
	}()

	// if err := setReuseAddress(fd); err != nil {
	// 	return nil, err
	// }

	bindAddr, err := wasip1syscall.NetaddrToRaw(addr)
	if err != nil {
		return nil, os.NewSyscallError("bind", err)
	}

	if err := wasip1syscall.Bind(fd, bindAddr); err != nil {
		return nil, os.NewSyscallError("bind", err)
	}

	const backlog = 64 // TODO: configurable?
	if err := wasip1syscall.Listen(fd, backlog); err != nil {
		return nil, os.NewSyscallError("listen", err)
	}

	name, err := wasip1syscall.Getsockname(fd)
	if err != nil {
		return nil, os.NewSyscallError("getsockname", err)
	}

	log.Println("listeners not yet supported", name)
	f := wasip1net.Socket(uintptr(fd))
	fd = -1 // now the *os.File owns the file descriptor
	defer f.Close()

	return nil, syscall.ENOTSUP
	// l, err := net.FileListener(f)
	// if err != nil {
	// 	return nil, err
	// }
	// return makeListener(l, name), nil
}

func listenPacketAddr(addr net.Addr) (net.PacketConn, error) {
	return nil, syscall.ENOTSUP
	// af := netaddrfamily(addr)
	// sotype, err := socketType(addr)
	// if err != nil {
	// 	return nil, os.NewSyscallError("socket", err)
	// }
	// fd, err := socket(af, syscall.SOCK_DGRAM, netaddrproto(addr))
	// if err != nil {
	// 	return nil, os.NewSyscallError("socket", err)
	// }
	// defer func() {
	// 	if fd >= 0 {
	// 		syscall.Close(fd)
	// 	}
	// }()

	// if err := setReuseAddress(fd); err != nil {
	// 	return nil, err
	// }

	// bindAddr, err := netaddrToSockaddr(addr)
	// if err != nil {
	// 	return nil, os.NewSyscallError("bind", err)
	// }
	// if err := bind(fd, bindAddr); err != nil {
	// 	return nil, os.NewSyscallError("bind", err)
	// }

	// name, err := getsockname(fd)
	// if err != nil {
	// 	return nil, os.NewSyscallError("getsockname", err)
	// }
	// laddr := sockipToNetAddr(af, sotype)(name)
	// if laddr == nil {
	// 	return nil, fmt.Errorf("unsupported address")
	// }
	// fd = -1 // now the *netFD owns the file descriptor
	// return wasip1net.PacketConn(uintptr(fd), af, sotype, socnetwork(af, sotype), laddr, nil)
}

type listener struct{ net.Listener }

func (l *listener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	return initconn(c)
}

// func sockaddrIPAndPort(addr sockaddr) (net.IP, int) {
// 	switch a := addr.(type) {
// 	case *sockipaddr[sockip4]:
// 		return net.IP(a.addr.ip[:]), int(a.port)
// 	case *sockipaddr[sockip6]:
// 		return net.IP(a.addr.ip[:]), int(a.port)
// 	default:
// 		return nil, 0
// 	}
// }

// func sockaddrName(addr sockaddr) string {
// 	switch a := addr.(type) {
// 	case *sockaddrUnix:
// 		return a.name
// 	default:
// 		return ""
// 	}
// }
