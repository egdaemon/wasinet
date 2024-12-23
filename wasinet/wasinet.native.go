//go:build !wasip1

package wasinet

import (
	"context"
	"encoding/binary"
	"log"
	"net"
	"syscall"
	"unsafe"

	"github.com/egdaemon/wasinetruntime/ffi"
	"github.com/egdaemon/wasinetruntime/ffiguest"
	"golang.org/x/sys/unix"
)

func unixsockaddr(addr unsafe.Pointer, addrLen uintptr) (sa unix.Sockaddr, err error) {
	v := ffiguest.RawRead[rawSockaddrAny](addr, addrLen)
	wsa, err := anyToSockaddr(&v)
	if err != nil {
		return nil, err
	}

	switch t := wsa.(type) {
	case *sockipaddr[sockip4]:
		log.Println("IPv4")
		return &unix.SockaddrInet4{Port: int(t.port), Addr: t.addr.ip}, nil
	case *sockipaddr[sockip6]:
		log.Println("IPv6")
		return &unix.SockaddrInet6{Port: int(t.port), Addr: t.addr.ip, ZoneId: 0}, nil
	case *sockaddrUnix:
		log.Println("Unix")
		return &unix.SockaddrUnix{Name: t.name}, nil
	default:
		return nil, syscall.ENOTSUP
	}
}

func wasisocketaddr(sa unix.Sockaddr) (*rawSockaddrAny, error) {
	switch t := sa.(type) {
	case *unix.SockaddrInet4:
		a := sockipaddr[sockip4]{port: uint32(t.Port), addr: sockip4{ip: t.Addr}}
		return a.sockaddr(), nil

	case *unix.SockaddrInet6:
		a := sockipaddr[sockip6]{port: uint32(t.Port), addr: sockip6{ip: t.Addr, zone: t.ZoneId}}
		return a.sockaddr(), nil
	case *unix.SockaddrUnix:
		name := t.Name
		if len(name) == 0 {
			// For consistency across platforms, replace empty unix socket
			// addresses with @. On Linux, addresses where the first byte is
			// a null byte are considered abstract unix sockets, and the first
			// byte is replaced with @.
			name = "@"
		}
		return (&sockaddrUnix{name: name}).sockaddr(), nil
	default:
		return nil, syscall.EINVAL
	}
}

func sock_open(af int32, socktype int32, proto int32, fd unsafe.Pointer) syscall.Errno {
	log.Println("sock_open", af, socktype, proto)
	_fd, errno := unix.Socket(int(af), int(socktype), int(proto))
	log.Println("opened", _fd)
	ffiguest.WriteInt32(fd, int32(_fd))
	return ffi.Errno(errno)
}

func sock_bind(fd int32, addr unsafe.Pointer, addrLen uintptr) syscall.Errno {
	wsa, err := unixsockaddr(addr, addrLen)
	if err != nil {
		return ffi.Errno(err)
	}

	log.Printf("YAY: %d - %T\n", fd, wsa)
	return ffi.Errno(unix.Bind(int(fd), wsa))
}

func sock_listen(fd int32, backlog int32) syscall.Errno {
	log.Println("listen", fd, backlog)
	err := ffi.Errno(unix.Listen(int(fd), int(backlog)))
	return err
}

func sock_connect(fd int32, addr unsafe.Pointer, addrlen uintptr, port uint32) syscall.Errno {
	return syscall.ENOTSUP
}

func sock_getsockopt(fd int32, level uint32, name uint32, value unsafe.Pointer, valueLen uint32) syscall.Errno {
	return syscall.ENOTSUP
}

func sock_setsockopt(fd int32, level uint32, name uint32, valueptr unsafe.Pointer, valueLen uintptr) syscall.Errno {
	switch name {
	case syscall.SO_LINGER: // TODO: implement SO_LINGER
		// This accepts a struct linger value.
		return syscall.ENOTSUP
	case syscall.SO_BINDTODEVICE: // TODO: implement SO_BINDTODEVICE
		// This accepts a string value.
		return syscall.ENOTSUP
	default:
		value := binary.LittleEndian.Uint32(ffiguest.BytesRead(valueptr, valueLen))
		return ffi.Errno(unix.SetsockoptInt(int(fd), int(level), int(name), int(value)))
	}
}

func sock_getlocaladdr(fd int32, addr unsafe.Pointer) syscall.Errno {
	log.Println("sock_getlocaladdr")
	sa, err := unix.Getsockname(int(fd))
	if err != nil {
		return ffi.Errno(err)
	}
	_addr, err := wasisocketaddr(sa)
	if err != nil {
		return ffi.Errno(err)
	}
	log.Println(_addr)
	ffiguest.WriteRaw(addr, *_addr)
	return ffi.ErrnoSuccess()
}

func sock_getpeeraddr(fd int32, addr unsafe.Pointer) syscall.Errno {
	return syscall.ENOTSUP
}

func sock_recv_from(
	fd int32,
	iovs unsafe.Pointer,
	iovsCount int32,
	addr unsafe.Pointer,
	iflags int32,
	port unsafe.Pointer,
	nread unsafe.Pointer,
	oflags unsafe.Pointer,
) syscall.Errno {
	return syscall.ENOTSUP
}

func sock_send_to(
	fd int32,
	iovs unsafe.Pointer,
	iovsCount int32,
	addr unsafe.Pointer,
	port int32,
	flags int32,
	nwritten unsafe.Pointer,
) syscall.Errno {
	return syscall.ENOTSUP
}

func sock_getaddrip(
	networkptr unsafe.Pointer, networklen uint32,
	addressptr unsafe.Pointer, addresslen uint32,
	ipres unsafe.Pointer, maxResLen uint32, ipreslen unsafe.Pointer,
) syscall.Errno {
	var (
		err error
		ip  []net.IP
		buf []byte
	)
	network := unsafe.String((*byte)(networkptr), networklen)
	address := unsafe.String((*byte)(addressptr), addresslen)
	if ip, err = net.DefaultResolver.LookupIP(context.Background(), network, address); err != nil {
		log.Println("sock_getaddrip lookup failed", err)
		return syscall.EINVAL
	}

	reslength := len(ip)
	if reslength*net.IPv6len > int(maxResLen) {
		reslength = int(maxResLen / net.IPv6len)
	}
	log.Println("DERP RESLENGTH", reslength)

	buf = make([]byte, 0, maxResLen)
	for _, i := range ip[:reslength] {
		buf = append(buf, i.To16()...)
	}

	*(*unsafe.Pointer)(ipres) = unsafe.Pointer(&buf[0])
	*(*uint32)(ipreslen) = uint32(len(buf))

	return 0
}

func sock_getaddrport(
	networkptr unsafe.Pointer, networklen uint32,
	serviceptr unsafe.Pointer, servicelen uint32,
	portptr unsafe.Pointer,
) uint32 {
	var (
		err  error
		port int
	)

	network := ffiguest.StringRead(networkptr, networklen)
	service := ffiguest.StringRead(serviceptr, servicelen)
	log.Println("sock_getaddrport", network, service)
	if port, err = net.DefaultResolver.LookupPort(context.Background(), network, service); err != nil {
		return uint32(ffi.Errno(err))
	}

	ffiguest.WriteUint32(portptr, uint32(port))

	return 0
}

func sock_shutdown(fd, how int32) syscall.Errno {
	return syscall.ENOTSUP
}
