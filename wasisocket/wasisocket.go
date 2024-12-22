package wasisocket

import (
	"context"
	"log"
	"syscall"

	"github.com/tetratelabs/wazero/api"
	"golang.org/x/sys/unix"
)

type OpenFn func(ctx context.Context, domain, utype, protocol int) (int32, uint16)
type OpenHostFn func(ctx context.Context, m api.Module, domain int32, family int32, proto int32, fd uintptr) uint32

func Open(open OpenFn) OpenHostFn {
	return func(
		ctx context.Context,
		m api.Module,
		domain int32,
		family int32,
		proto int32,
		fd uintptr,
	) uint32 {
		log.Println("socket_open")
		return uint32(syscall.EACCES)
	}
}

type BindFn func(ctx context.Context, fd int, sa unix.Sockaddr) error
type BindHostFn func(ctx context.Context, m api.Module, fd int32, addr uintptr, addrlen uint32) uint32

func Bind(bind BindFn) BindHostFn {
	return func(
		ctx context.Context,
		m api.Module,
		fd int32,
		addr uintptr,
		addrlen uint32,
	) uint32 {
		log.Println("socket_bind")
		return uint32(syscall.EACCES)
	}
}

type ConnectFn func(ctx context.Context, fd int, sa unix.Sockaddr) error
type ConnectHostFn func(ctx context.Context, m api.Module, fd int32, addr uintptr, addrlen uint32) uint32

func Connect(fn ConnectFn) ConnectHostFn {
	return func(
		ctx context.Context,
		m api.Module,
		fd int32,
		addr uintptr,
		addrlen uint32,
	) uint32 {
		log.Println("socket_connect")
		return uint32(syscall.EACCES)
	}
}

type ListenFn func(ctx context.Context, fd int, backlog int) error
type ListenHostFn func(ctx context.Context, m api.Module, fd int32, backlog int32) uint32

func Listen(fn ListenFn) ListenHostFn {
	return func(
		ctx context.Context,
		m api.Module,
		fd int32,
		backlog int32,
	) uint32 {
		log.Println("socket_listen")
		return uint32(syscall.EACCES)
	}
}

type SendToFn func(ctx context.Context, fd int, buf []byte, flags int, to unix.Sockaddr) (int, error)
type SendToHostFn func(ctx context.Context, m api.Module, fd int32, buf uintptr, len uint32, flags int32, addr uintptr, addrlen uint32) uint32

func SendTo(fn SendToFn) SendToHostFn {
	return func(
		ctx context.Context,
		m api.Module,
		fd int32,
		buf uintptr,
		len uint32,
		flags int32,
		addr uintptr,
		addrlen uint32,
	) uint32 {
		log.Println("socket_send_to")
		return uint32(syscall.EACCES)
	}
}

type RecvFromFn func(ctx context.Context, fd int, buf []byte, flags int, from unix.Sockaddr) (int, unix.Sockaddr, error)
type RecvFromHostFn func(ctx context.Context, m api.Module, fd int32, buf uintptr, len uint32, flags int32, addr uintptr, addrlen uintptr) uint32

func RecvFrom(fn RecvFromFn) RecvFromHostFn {
	return func(
		ctx context.Context,
		m api.Module,
		fd int32,
		buf uintptr,
		len uint32,
		flags int32,
		addr uintptr,
		addrlen uintptr,
	) uint32 {
		log.Println("socket_recv_from")
		return uint32(syscall.EACCES)
	}
}

type SetOptFn func(ctx context.Context, fd int, level, name int, value []byte) error
type SetOptHostFn func(ctx context.Context, m api.Module, fd int32, level int32, name int32, value uintptr, vallen uint32) uint32

func SetOpt(fn SetOptFn) SetOptHostFn {
	return func(
		ctx context.Context,
		m api.Module,
		fd int32,
		level int32,
		name int32,
		value uintptr,
		vallen uint32,
	) uint32 {
		log.Println("socket_set_opt")
		return uint32(syscall.EACCES)
	}
}

type GetOptFn func(ctx context.Context, fd int, level, name int, value []byte) (int, error)
type GetOptHostFn func(ctx context.Context, m api.Module, fd int32, level int32, name int32, value uintptr, vallen uint32) uint32

func GetOpt(fn GetOptFn) GetOptHostFn {
	return func(
		ctx context.Context,
		m api.Module,
		fd int32,
		level int32,
		name int32,
		value uintptr,
		vallen uint32,
	) uint32 {
		log.Println("socket_get_opt")
		return uint32(syscall.EACCES)
	}
}

type LocalAddrFn func(ctx context.Context, fd int) (unix.Sockaddr, error)
type LocalAddrHostFn func(ctx context.Context, m api.Module, fd int32, addr uintptr, addrlen uintptr) uint32

func LocalAddr(fn LocalAddrFn) LocalAddrHostFn {
	return func(
		ctx context.Context,
		m api.Module,
		fd int32,
		addr uintptr,
		addrlen uintptr,
	) uint32 {
		log.Println("socket_local_addr")
		return uint32(syscall.EACCES)
	}
}

type PeerAddrFn func(ctx context.Context, fd int) (unix.Sockaddr, error)
type PeerAddrHostFn func(ctx context.Context, m api.Module, fd int32, addr uintptr, addrlen uintptr) uint32

func PeerAddr(fn PeerAddrFn) PeerAddrHostFn {
	return func(
		ctx context.Context,
		m api.Module,
		fd int32,
		addr uintptr,
		addrlen uintptr,
	) uint32 {
		log.Println("socket_peer_addr")
		return uint32(syscall.EACCES)
	}
}

type AddrInfoFn func(ctx context.Context, domain, utype, protocol int, addr, port string) ([]unix.Sockaddr, error)
type AddrInfoHostFn func(ctx context.Context, m api.Module, domain int32, utype int32, protocol int32, addr uintptr, port uintptr) uint32

func AddrInfo(fn AddrInfoFn) AddrInfoHostFn {
	return func(
		ctx context.Context,
		m api.Module,
		domain int32,
		utype int32,
		protocol int32,
		addr uintptr,
		port uintptr,
	) uint32 {
		log.Println("socket_addr_info")
		return uint32(syscall.EACCES)
	}
}

// "sock_open":         wazergo.F3((*Module).WasmEdgeSockOpen),
// "sock_bind":         wazergo.F3((*Module).WasmEdgeSockBind),
// "sock_connect":      wazergo.F3((*Module).WasmEdgeSockConnect),
// "sock_listen":       wazergo.F2((*Module).WasmEdgeSockListen),
// "sock_send_to":      wazergo.F6((*Module).WasmEdgeSockSendTo),
// "sock_recv_from":    wazergo.F7((*Module).WasmEdgeV2SockRecvFrom),
// "sock_getsockopt":   wazergo.F5((*Module).WasmEdgeSockGetOpt),
// "sock_setsockopt":   wazergo.F4((*Module).WasmEdgeSockSetOpt),
// "sock_getlocaladdr": wazergo.F3((*Module).WasmEdgeV2SockLocalAddr),
// "sock_getpeeraddr":  wazergo.F3((*Module).WasmEdgeV2SockPeerAddr),
// "sock_getaddrinfo":  wazergo.F6((*Module).WasmEdgeSockAddrInfo),
