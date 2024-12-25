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
	wsa, err := ffi.RawRead[rawsocketaddr](m, addr, addrlen)
	if err != nil {
		return nil, err
	}

	return unixsockaddr(wsa)
}

type OpenFn func(ctx context.Context, af, socktype, protocol int) (fd int, err error)
type OpenHostFn func(ctx context.Context, m ffi.Memory, af uint32, socktype uint32, proto uint32, fd uintptr) syscall.Errno

func SocketOpen(open OpenFn) OpenHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		af uint32, socktype uint32, proto uint32, fd uintptr,
	) syscall.Errno {
		log.Println("socket open initated")
		defer log.Println("socket open completed")
		_fd, errno := open(ctx, int(af), int(socktype), int(proto))
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
		log.Println("socket bind initated")
		defer log.Println("socket bind completed")
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
		log.Println("socket connect initated")
		defer log.Println("socket connect completed")
		sa, err := readsockaddr(m, addr, addrlen)
		if err != nil {
			return ffi.Errno(err)
		}
		return ffi.Errno(fn(ctx, int(fd), sa))
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
		return ffi.Errno(fn(ctx, int(fd), int(backlog)))
	}
}

type SendToFn func(ctx context.Context, fd int, sa unix.Sockaddr, vecs [][]byte, flags int) (int, error)
type SendToHostFn func(
	ctx context.Context,
	m ffi.Memory,
	fd int32,
	iovs uintptr, iovslen uint32,
	addr uintptr, addrlen uint32,
	flags int32,
	nwritten uintptr,
) syscall.Errno

func SocketSendTo(fn SendToFn) SendToHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		fd int32,
		iovs uintptr, iovslen uint32,
		addrptr uintptr, addrlen uint32,
		flags int32,
		nwritten uintptr,
	) syscall.Errno {
		log.Println("socket send_to initated")
		defer log.Println("socket send_to completed")

		vecs, err := ffi.ReadSlice[[]byte](m, iovs, iovslen)
		if err != nil {
			return ffi.Errno(err)
		}
		sa, err := readsockaddr(m, addrptr, addrlen)
		if err != nil {
			return ffi.Errno(err)
		}
		log.Println("socket_send_to is not implemented", len(vecs), len(vecs[0]))
		n, err := fn(ctx, int(fd), sa, vecs, int(flags))
		if err != nil {
			return ffi.Errno(err)
		}

		if err = ffi.Uint32Write(m, nwritten, uint32(n)); err != nil {
			return ffi.Errno(err)
		}

		return ffi.ErrnoSuccess()
	}
}

type RecvFromFn func(ctx context.Context, fd int, buf [][]byte, flags int) (int, int, unix.Sockaddr, error)
type RecvFromHostFn func(
	ctx context.Context,
	m ffi.Memory,
	fd int32,
	iovs uintptr, iovslen uint32,
	addrptr uintptr,
	iflags int32,
	nread uintptr,
	oflags uintptr,
) syscall.Errno

func SocketRecvFrom(fn RecvFromFn) RecvFromHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		fd int32,
		iovs uintptr, iovslen uint32,
		addrptr uintptr,
		iflags int32,
		nread uintptr,
		oflags uintptr,
	) syscall.Errno {
		log.Println("socket recv_from initated")
		defer log.Println("socket recv_from completed")
		vecs, err := ffi.ReadSlice[[]byte](m, iovs, iovslen)
		if err != nil {
			return ffi.Errno(err)
		}

		log.Println("socket_recv_from is not implemented", len(vecs), len(vecs[0]))
		n, roflags, sa, err := fn(ctx, int(fd), vecs, int(iflags))
		if err != nil {
			return ffi.Errno(err)
		}

		addr, err := unixsockaddrToRaw(sa)
		if err != nil {
			return ffi.Errno(err)
		}

		if err = ffi.RawWrite(m, addr, addrptr, 0); err != nil {
			return ffi.Errno(err)
		}

		if err = ffi.Uint32Write(m, nread, uint32(n)); err != nil {
			return ffi.Errno(err)
		}

		if err = ffi.Uint32Write(m, oflags, uint32(roflags)); err != nil {
			return ffi.Errno(err)
		}

		return ffi.ErrnoSuccess()
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
		valuelen uint32,
	) syscall.Errno {
		log.Println("socket setopt initated")
		defer log.Println("socket setopt completed")
		value, err := ffi.BytesRead(m, valueptr, valuelen)
		if err != nil {
			return ffi.Errno(err)
		}

		return ffi.Errno(fn(ctx, int(fd), int(level), int(name), value))
	}
}

type GetOptFn func(ctx context.Context, fd int, level, name int, value []byte) (any, error)
type GetOptHostFn func(ctx context.Context, m ffi.Memory, fd int32, level int32, name int32, value uintptr, vallen uint32) syscall.Errno

func SocketGetOpt(fn GetOptFn) GetOptHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		fd int32,
		level int32,
		name int32,
		valueptr uintptr,
		valuelen uint32,
	) syscall.Errno {
		log.Println("socket getopt initated")
		defer log.Println("socket getopt completed")
		value, err := ffi.BytesRead(m, valueptr, valuelen)
		if err != nil {
			return ffi.Errno(err)
		}
		rv, err := fn(ctx, int(fd), int(level), int(name), value)
		if err != nil {
			return ffi.Errno(err)
		}
		switch av := rv.(type) {
		case int:
			return ffi.Errno(ffi.Uint32Write(m, valueptr, uint32(av)))
		default:
			log.Printf("unsupported socket option type: %T\n", rv)
			return syscall.ENOTSUP
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
		log.Println("socket localaddr initated")
		defer log.Println("socket localaddr completed")
		sa, err := unix.Getsockname(int(fd))
		if err != nil {
			return ffi.Errno(err)
		}
		addr, err := unixsockaddrToRaw(sa)
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
		log.Println("socket peeraddr initated")
		defer log.Println("socket peeraddr completed")
		sa, err := unix.Getpeername(int(fd))
		if err != nil {
			return ffi.Errno(err)
		}
		addr, err := unixsockaddrToRaw(sa)
		if err != nil {
			return ffi.Errno(err)
		}
		return ffi.Errno(ffi.RawWrite(m, addr, addrptr, addrlen))
	}
}

type ShutdownFn func(ctx context.Context, fd, how int) error
type ShutdownHostFn func(ctx context.Context, m ffi.Memory, fd, how int32) syscall.Errno

func SocketShutdown(fn ShutdownFn) ShutdownHostFn {
	return func(
		ctx context.Context, m ffi.Memory, fd, how int32,
	) syscall.Errno {
		log.Println("socket shutdown initated")
		defer log.Println("socket shutdown completed")
		return ffi.Errno(fn(ctx, int(fd), int(how)))
	}
}

type AddrPortFn func(ctx context.Context, network string, service string) (int, error)
type AddrPortHostFn func(ctx context.Context,
	m ffi.Memory,
	networkptr uintptr, networklen uint32,
	serviceptr uintptr, servicelen uint32,
	portptr uintptr,
) syscall.Errno

func SocketAddrPort(portfn AddrPortFn) AddrPortHostFn {
	return func(
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

		log.Println("socket addrport initated")
		defer log.Println("socket addport completed")

		network, err := ffi.ReadString(m, networkptr, networklen)
		if err != nil {
			return ffi.Errno(err)
		}
		service, err := ffi.ReadString(m, serviceptr, servicelen)
		if err != nil {
			return ffi.Errno(err)
		}

		if port, err = portfn(ctx, network, service); err != nil {
			return ffi.Errno(err)
		}

		return ffi.Errno(ffi.Uint32Write(m, portptr, uint32(port)))
	}
}

type AddrIPFn func(ctx context.Context, network string, address string) ([]net.IP, error)
type AddrIPHostFn func(
	ctx context.Context,
	m ffi.Memory,
	networkptr uintptr, networklen uint32,
	addressptr uintptr, addresslen uint32,
	ipres uintptr, maxResLen uint32,
	ipreslen uintptr,
) syscall.Errno

func SocketAddrIP(fn AddrIPFn) AddrIPHostFn {
	return func(
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

		log.Println("socket addrip initated")
		defer log.Println("socket addrip completed")

		network, err := ffi.ReadString(m, networkptr, networklen)
		if err != nil {
			return ffi.Errno(err)
		}
		address, err := ffi.ReadString(m, addressptr, addresslen)
		if err != nil {
			return ffi.Errno(err)
		}

		if ip, err = fn(ctx, network, address); err != nil {
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
}
