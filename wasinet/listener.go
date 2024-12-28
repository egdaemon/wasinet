package wasinet

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"syscall"

	"github.com/egdaemon/wasinet/wasinet/internal/errorsx"
)

// Listen announces on the local network address.
func Listen(network, address string) (net.Listener, error) {
	switch network {
	case "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6", "unix":
	default:
		return nil, unsupportedNetwork(network, address)
	}
	addrs, err := lookupAddr(context.Background(), oplisten, network, address)
	if err != nil {
		return nil, netOpErr(oplisten, unresolvedaddr(network, address), err)
	}

	firstaddr := addrs[0]
	lstn, err := listenAddr(firstaddr)
	return lstn, netOpErr(oplisten, firstaddr, err)
}

// ListenPacket creates a listening packet connection.
func ListenPacket(network, address string) (net.PacketConn, error) {
	switch network {
	case "udp", "udp4", "udp6", "unixgram":
	default:
		return nil, unsupportedNetwork(network, address)
	}
	addrs, err := lookupAddr(context.Background(), oplisten, network, address)
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
	fd, err := socket(netaddrfamily(addr), syscall.SOCK_STREAM, netaddrproto(addr))
	if err != nil {
		return nil, os.NewSyscallError("socket", err)
	}
	defer func() {
		if fd >= 0 {
			syscall.Close(fd)
		}
	}()

	if err := setReuseAddress(fd); err != nil {
		return nil, err
	}

	bindAddr, err := netaddrToSockaddr(addr)
	if err != nil {
		return nil, os.NewSyscallError("bind", err)
	}

	if err := bind(fd, bindAddr); err != nil {
		return nil, os.NewSyscallError("bind", err)
	}

	const backlog = 64 // TODO: configurable?
	if err := listen(fd, backlog); err != nil {
		return nil, os.NewSyscallError("listen", err)
	}

	name, err := getsockname(fd)
	if err != nil {
		return nil, os.NewSyscallError("getsockname", err)
	}

	f := os.NewFile(uintptr(fd), "")
	fd = -1 // now the *os.File owns the file descriptor
	defer f.Close()

	l, err := net.FileListener(f)
	if err != nil {
		return nil, err
	}
	return makeListener(l, name), nil
}

func listenPacketAddr(addr net.Addr) (net.PacketConn, error) {
	af := netaddrfamily(addr)
	sotype, err := socketType(addr)
	if err != nil {
		return nil, os.NewSyscallError("socket", err)
	}
	fd, err := socket(af, syscall.SOCK_DGRAM, netaddrproto(addr))
	if err != nil {
		return nil, os.NewSyscallError("socket", err)
	}
	defer func() {
		if fd >= 0 {
			syscall.Close(fd)
		}
	}()

	if err := setReuseAddress(fd); err != nil {
		return nil, err
	}

	bindAddr, err := netaddrToSockaddr(addr)
	if err != nil {
		return nil, os.NewSyscallError("bind", err)
	}
	if err := bind(fd, bindAddr); err != nil {
		return nil, os.NewSyscallError("bind", err)
	}

	name, err := getsockname(fd)
	if err != nil {
		return nil, os.NewSyscallError("getsockname", err)
	}
	laddr := sockipToNetAddr(af, sotype)(name)
	if laddr == nil {
		return nil, fmt.Errorf("unsupported address")
	}
	sconn := newFD(fd, af, sotype, socnetwork(af, sotype), laddr, nil)
	fd = -1 // now the *netFD owns the file descriptor
	return makePacketConn(sconn), nil
}

type listener struct{ net.Listener }

func (l *listener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	return makeConn(c)
}

type unixListener struct {
	listener
	addr net.UnixAddr
}

func (l *unixListener) Addr() net.Addr {
	return &l.addr
}

func makeListener(l net.Listener, addr sockaddr) net.Listener {
	switch addr.(type) {
	case *sockaddrUnix:
		l = &unixListener{listener: listener{l}}
	default:
		l = &listener{l}
	}
	setNetAddr(syscall.SOCK_STREAM, l.Addr(), addr)
	return l
}

func setNetAddr(sotype int, dst net.Addr, src sockaddr) {
	switch a := dst.(type) {
	case *net.IPAddr:
		a.IP, _ = sockaddrIPAndPort(src)
	case *net.TCPAddr:
		a.IP, a.Port = sockaddrIPAndPort(src)
	case *net.UDPAddr:
		a.IP, a.Port = sockaddrIPAndPort(src)
	case *net.UnixAddr:
		switch sotype {
		case syscall.SOCK_STREAM:
			a.Net = "unix"
		case syscall.SOCK_DGRAM:
			a.Net = "unixgram"
		}
		a.Name = sockaddrName(src)
	default:
		log.Printf("unable to set addr: %T\n", dst)
	}
}

// In Go 1.21, the net package cannot initialize the local and remote addresses
// of network connections. For this reason, we use this function to retreive the
// addresses and return a wrapped net.Conn with LocalAddr/RemoteAddr implemented.
func makeConn(c net.Conn) (x net.Conn, err error) {
	syscallConn, ok := c.(syscall.Conn)
	if !ok {
		return c, nil
	}

	defer func() {
		if err == nil {
			return
		}

		if c != nil {
			log.Println("CLOSING", err)
			c.Close()
		}
	}()

	rawConn, err := syscallConn.SyscallConn()
	if err != nil {
		return nil, fmt.Errorf("syscall.Conn.SyscallConn: %w", err)
	}
	cerr := rawConn.Control(func(fd uintptr) {
		var (
			addr sockaddr
			peer sockaddr
		)

		if addr, err = getsockname(int(fd)); err != nil {
			err = os.NewSyscallError("getsockname", err)
			return
		}

		if peer, err = getpeername(int(fd)); err != nil {
			err = os.NewSyscallError("getpeername", err)
			return
		}

		setNetAddr(syscall.SOCK_STREAM, c.LocalAddr(), addr)
		setNetAddr(syscall.SOCK_STREAM, c.RemoteAddr(), peer)

		if a, ok := c.(*netFD); ok {
			a.initremote()
		}

		if _, unix := addr.(*sockaddrUnix); unix {
			c = &unixConn{Conn: c}
		}
	})

	return c, errorsx.Compact(err, cerr)
}

func sockaddrIPAndPort(addr sockaddr) (net.IP, int) {
	switch a := addr.(type) {
	case *sockipaddr[sockip4]:
		return net.IP(a.addr.ip[:]), int(a.port)
	case *sockipaddr[sockip6]:
		return net.IP(a.addr.ip[:]), int(a.port)
	default:
		return nil, 0
	}
}

func sockaddrName(addr sockaddr) string {
	switch a := addr.(type) {
	case *sockaddrUnix:
		return a.name
	default:
		return ""
	}
}
