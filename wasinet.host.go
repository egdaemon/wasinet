//go:build !wasip1

package wasinet

import (
	"context"
	"log"
	"net"
	"syscall"

	"github.com/egdaemon/wasinet/ffi"
	"golang.org/x/sys/unix"
)

func readsockaddr(
	m ffi.Memory, addr uintptr, addrlen uint32,
) (unix.Sockaddr, error) {
	wsa, err := ffi.RawRead[rawSockaddrAny](m, addr, addrlen)
	if err != nil {
		return nil, err
	}

	return unixsockaddr(wsa)
}

type OpenFn func(ctx context.Context, af, socktype, protocol uint32) (fd int, err error)
type OpenHostFn func(ctx context.Context, m ffi.Memory, af uint32, socktype uint32, proto uint32, fd uintptr) syscall.Errno

func SocketOpen(open OpenFn) OpenHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		af uint32, socktype uint32, proto uint32, fd uintptr,
	) syscall.Errno {
		_fd, errno := open(ctx, af, socktype, proto)
		if !m.WriteUint32Le(uint32(fd), uint32(_fd)) {
			return syscall.EFAULT
		}

		return ffi.Errno(errno)
	}
}

type BindFn func(ctx context.Context, fd int, sa unix.Sockaddr) error
type BindHostFn func(ctx context.Context, m ffi.Memory, fd uint32, addr uintptr, addrlen uint32) syscall.Errno

func SocketBind(bind BindFn) BindHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		fd uint32,
		addr uintptr,
		addrlen uint32,
	) syscall.Errno {
		sa, err := readsockaddr(m, addr, addrlen)
		if err != nil {
			return ffi.Errno(err)
		}
		return ffi.Errno(bind(ctx, int(fd), sa))
	}
}

type ConnectFn func(ctx context.Context, fd int, sa unix.Sockaddr) error
type ConnectHostFn func(ctx context.Context, m ffi.Memory, fd int32, addr uintptr, addrlen uint32) syscall.Errno

func SocketConnect(fn ConnectFn) ConnectHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		fd int32,
		addr uintptr,
		addrlen uint32,
	) syscall.Errno {
		sa, err := readsockaddr(m, addr, addrlen)
		if err != nil {
			return ffi.Errno(err)
		}
		return ffi.Errno(unix.Connect(int(fd), sa))
	}
}

type ListenFn func(ctx context.Context, fd int, backlog int) error
type ListenHostFn func(ctx context.Context, m ffi.Memory, fd int32, backlog int32) syscall.Errno

func SocketListen(fn ListenFn) ListenHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		fd int32,
		backlog int32,
	) syscall.Errno {
		return ffi.Errno(unix.Listen(int(fd), int(backlog)))
	}
}

type SendToFn func(ctx context.Context, fd int, buf []byte, flags int, to unix.Sockaddr) (int, error)
type SendToHostFn func(ctx context.Context, m ffi.Memory, fd int32, buf uintptr, len uint32, flags int32, addr uintptr, addrlen uint32) syscall.Errno

func SocketSendTo(fn SendToFn) SendToHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		fd int32,
		buf uintptr,
		len uint32,
		flags int32,
		addr uintptr,
		addrlen uint32,
	) syscall.Errno {
		log.Println("socket_send_to is not implemented")
		return syscall.ENOTSUP
	}
}

type RecvFromFn func(ctx context.Context, fd int, buf []byte, flags int, from unix.Sockaddr) (int, unix.Sockaddr, error)
type RecvFromHostFn func(ctx context.Context, m ffi.Memory, fd int32, buf uintptr, len uint32, flags int32, addr uintptr, addrlen uintptr) syscall.Errno

func SocketRecvFrom(fn RecvFromFn) RecvFromHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		fd int32,
		buf uintptr,
		len uint32,
		flags int32,
		addr uintptr,
		addrlen uintptr,
	) syscall.Errno {
		log.Println("socket_recv_from is not implemented")
		return syscall.ENOTSUP
	}
}

type SetOptFn func(ctx context.Context, fd int, level, name int, value []byte) error
type SetOptHostFn func(ctx context.Context, m ffi.Memory, fd int32, level int32, name int32, value uintptr, vallen uint32) syscall.Errno

func SocketSetOpt(fn SetOptFn) SetOptHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		fd int32,
		level int32,
		name int32,
		valueptr uintptr,
		valueLen uint32,
	) syscall.Errno {
		switch name {
		case syscall.SO_LINGER: // this is untested.
			value, err := ffi.RawRead[unix.Timeval](m, valueptr, valueLen)
			if err != nil {
				return ffi.Errno(err)
			}
			return ffi.Errno(unix.SetsockoptTimeval(int(fd), int(level), int(name), value))
		case syscall.SO_BINDTODEVICE: // this is untested.
			value, err := ffi.ReadString(m, valueptr, valueLen)
			if err != nil {
				return ffi.Errno(err)
			}
			return ffi.Errno(unix.SetsockoptString(int(fd), int(level), int(name), value))
		default:
			value, err := ffi.Uint32Read(m, valueptr, valueLen)
			if err != nil {
				return ffi.Errno(err)
			}
			return ffi.Errno(unix.SetsockoptInt(int(fd), int(level), int(name), int(value)))
		}
	}
}

type GetOptFn func(ctx context.Context, fd int, level, name int, value []byte) (int, error)
type GetOptHostFn func(ctx context.Context, m ffi.Memory, fd int32, level int32, name int32, value uintptr, vallen uint32) syscall.Errno

func SocketGetOpt(fn GetOptFn) GetOptHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		fd int32,
		level int32,
		name int32,
		valueptr uintptr,
		vallen uint32,
	) syscall.Errno {
		switch name {
		default:
			v, err := unix.GetsockoptInt(int(fd), int(level), int(name))
			if err != nil {
				return ffi.Errno(err)
			}
			return ffi.Errno(ffi.Uint32Write(m, valueptr, uint32(v)))
		}
	}
}

type LocalAddrFn func(ctx context.Context, fd int) (unix.Sockaddr, error)
type LocalAddrHostFn func(ctx context.Context, m ffi.Memory, fd int32, addr uintptr, addrlen uint32) syscall.Errno

func SocketLocalAddr(fn LocalAddrFn) LocalAddrHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		fd int32,
		addrptr uintptr,
		addrlen uint32,
	) syscall.Errno {
		sa, err := unix.Getsockname(int(fd))
		if err != nil {
			return ffi.Errno(err)
		}
		addr, err := wasisocketaddr(sa)
		if err != nil {
			return ffi.Errno(err)
		}
		return ffi.Errno(ffi.RawWrite(m, addr, addrptr, addrlen))
	}
}

type PeerAddrFn func(ctx context.Context, fd int) (unix.Sockaddr, error)
type PeerAddrHostFn func(ctx context.Context, m ffi.Memory, fd int32, addr uintptr, addrlen uint32) syscall.Errno

func SocketPeerAddr(fn PeerAddrFn) PeerAddrHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		fd int32,
		addrptr uintptr,
		addrlen uint32,
	) syscall.Errno {
		sa, err := unix.Getpeername(int(fd))
		if err != nil {
			return ffi.Errno(err)
		}
		addr, err := wasisocketaddr(sa)
		if err != nil {
			return ffi.Errno(err)
		}
		return ffi.Errno(ffi.RawWrite(m, addr, addrptr, addrlen))
	}
}

type ShutdownFn func(ctx context.Context, fd, how int32) error
type ShutdownHostFn func(ctx context.Context, m ffi.Memory, fd, how int32) syscall.Errno

func SocketShutdown(fn ShutdownFn) ShutdownHostFn {
	return func(
		ctx context.Context, m ffi.Memory, fd, how int32,
	) syscall.Errno {
		return ffi.Errno(unix.Shutdown(int(fd), int(how)))
	}
}

func SocketAddrPort(
	ctx context.Context,
	m ffi.Memory,
	networkptr uintptr, networklen uint32,
	serviceptr uintptr, servicelen uint32,
	portptr uintptr,
) syscall.Errno {
	var (
		err  error
		port int
	)

	network, err := ffi.ReadString(m, networkptr, networklen)
	if err != nil {
		return ffi.Errno(err)
	}
	service, err := ffi.ReadString(m, serviceptr, servicelen)
	if err != nil {
		return ffi.Errno(err)
	}

	if port, err = net.DefaultResolver.LookupPort(ctx, network, service); err != nil {
		return ffi.Errno(err)
	}

	return ffi.Errno(ffi.Uint32Write(m, portptr, uint32(port)))
}

func SocketAddrIP(
	ctx context.Context,
	m ffi.Memory,
	networkptr uintptr, networklen uint32,
	addressptr uintptr, addresslen uint32,
	ipres uintptr, maxResLen uint32,
	ipreslen uintptr,
) syscall.Errno {
	var (
		err error
		ip  []net.IP
		buf []byte
	)
	network, err := ffi.ReadString(m, networkptr, networklen)
	if err != nil {
		return ffi.Errno(err)
	}
	address, err := ffi.ReadString(m, addressptr, addresslen)
	if err != nil {
		return ffi.Errno(err)
	}

	if ip, err = net.DefaultResolver.LookupIP(ctx, network, address); err != nil {
		log.Println("socket ip lookup failed", err)
		return syscall.EINVAL
	}

	reslength := len(ip)
	if reslength*net.IPv6len > int(maxResLen) {
		reslength = int(maxResLen / net.IPv6len)
	}

	buf = make([]byte, 0, maxResLen)
	for _, i := range ip[:reslength] {
		buf = append(buf, i.To16()...)
	}

	if err = ffi.BytesWrite(m, buf, ipres, maxResLen); err != nil {
		return ffi.Errno(err)
	}

	return ffi.Errno(ffi.Uint32Write(m, ipreslen, uint32(len(buf))))
}
