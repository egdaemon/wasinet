//go:build !wasip1

package wasinetruntime

import (
	"context"
	"log"
	"syscall"
	"unsafe"

	"github.com/egdaemon/wasinet/ffi"
	"golang.org/x/sys/unix"
)

type OpenFn func(ctx context.Context, af, socktype, protocol int32) (int32, syscall.Errno)
type OpenHostFn func(ctx context.Context, m ffi.Memory, af int32, socktype int32, proto int32, fd uintptr) syscall.Errno

func Open(open OpenFn) OpenHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		af int32, socktype int32, proto int32, fd uintptr,
	) syscall.Errno {
		_fd, errno := open(ctx, af, socktype, proto)
		if errno != 0 {
			return errno
		}

		if !m.WriteUint32Le(uint32(fd), uint32(_fd)) {
			return syscall.EFAULT
		}

		return ffi.Errno(errno)
	}
}

type BindFn func(ctx context.Context, fd int, sa unix.Sockaddr) error
type BindHostFn func(ctx context.Context, m ffi.Memory, fd int32, addr unsafe.Pointer, addrlen uint32) uint32

func Bind(bind BindFn) BindHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		fd int32,
		addr unsafe.Pointer,
		addrlen uint32,
	) uint32 {
		log.Println("socket_bind", addrlen)

		return uint32(syscall.ENOTSUP)
	}
}

type ConnectFn func(ctx context.Context, fd int, sa unix.Sockaddr) error
type ConnectHostFn func(ctx context.Context, m ffi.Memory, fd int32, addr uintptr, addrlen uint32) uint32

func Connect(fn ConnectFn) ConnectHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		fd int32,
		addr uintptr,
		addrlen uint32,
	) uint32 {
		log.Println("socket_connect")
		return uint32(syscall.ENOTSUP)
	}
}

type ListenFn func(ctx context.Context, fd int, backlog int) error
type ListenHostFn func(ctx context.Context, m ffi.Memory, fd int32, backlog int32) uint32

func Listen(fn ListenFn) ListenHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		fd int32,
		backlog int32,
	) uint32 {
		log.Println("socket_listen")
		return uint32(syscall.ENOTSUP)
	}
}

type SendToFn func(ctx context.Context, fd int, buf []byte, flags int, to unix.Sockaddr) (int, error)
type SendToHostFn func(ctx context.Context, m ffi.Memory, fd int32, buf uintptr, len uint32, flags int32, addr uintptr, addrlen uint32) uint32

func SendTo(fn SendToFn) SendToHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		fd int32,
		buf uintptr,
		len uint32,
		flags int32,
		addr uintptr,
		addrlen uint32,
	) uint32 {
		log.Println("socket_send_to")
		return uint32(syscall.ENOTSUP)
	}
}

type RecvFromFn func(ctx context.Context, fd int, buf []byte, flags int, from unix.Sockaddr) (int, unix.Sockaddr, error)
type RecvFromHostFn func(ctx context.Context, m ffi.Memory, fd int32, buf uintptr, len uint32, flags int32, addr uintptr, addrlen uintptr) uint32

func RecvFrom(fn RecvFromFn) RecvFromHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		fd int32,
		buf uintptr,
		len uint32,
		flags int32,
		addr uintptr,
		addrlen uintptr,
	) uint32 {
		log.Println("socket_recv_from")
		return uint32(syscall.ENOTSUP)
	}
}

type SetOptFn func(ctx context.Context, fd int, level, name int, value []byte) error
type SetOptHostFn func(ctx context.Context, m ffi.Memory, fd int32, level int32, name int32, value uintptr, vallen uint32) uint32

func SetOpt(fn SetOptFn) SetOptHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		fd int32,
		level int32,
		name int32,
		value uintptr,
		vallen uint32,
	) uint32 {
		log.Println("socket_set_opt")
		return uint32(syscall.ENOTSUP)
	}
}

type GetOptFn func(ctx context.Context, fd int, level, name int, value []byte) (int, error)
type GetOptHostFn func(ctx context.Context, m ffi.Memory, fd int32, level int32, name int32, value uintptr, vallen uint32) uint32

func GetOpt(fn GetOptFn) GetOptHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		fd int32,
		level int32,
		name int32,
		value uintptr,
		vallen uint32,
	) uint32 {
		log.Println("socket_get_opt")
		return uint32(syscall.ENOTSUP)
	}
}

type LocalAddrFn func(ctx context.Context, fd int) (unix.Sockaddr, error)
type LocalAddrHostFn func(ctx context.Context, m ffi.Memory, fd int32, addr uintptr, addrlen uintptr) uint32

func LocalAddr(fn LocalAddrFn) LocalAddrHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		fd int32,
		addr uintptr,
		addrlen uintptr,
	) uint32 {
		log.Println("socket_local_addr")
		return uint32(syscall.ENOTSUP)
	}
}

type PeerAddrFn func(ctx context.Context, fd int) (unix.Sockaddr, error)
type PeerAddrHostFn func(ctx context.Context, m ffi.Memory, fd int32, addr uintptr, addrlen uintptr) uint32

func PeerAddr(fn PeerAddrFn) PeerAddrHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		fd int32,
		addr uintptr,
		addrlen uintptr,
	) uint32 {
		log.Println("socket_peer_addr")
		return uint32(syscall.ENOTSUP)
	}
}

type AddrInfoFn func(ctx context.Context, domain, utype, protocol int, addr, port string) ([]unix.Sockaddr, error)
type AddrInfoHostFn func(ctx context.Context, m ffi.Memory, domain int32, utype int32, protocol int32, addr uintptr, port uintptr) uint32

func AddrInfo(fn AddrInfoFn) AddrInfoHostFn {
	return func(
		ctx context.Context,
		m ffi.Memory,
		domain int32,
		utype int32,
		protocol int32,
		addr uintptr,
		port uintptr,
	) uint32 {
		log.Println("socket_addr_info")
		return uint32(syscall.ENOTSUP)
	}
}
