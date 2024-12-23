//go:build wasip1

package wasinet

import (
	"syscall"
	"unsafe"
)

//go:wasmimport wasi_snapshot_preview1 sock_open
//go:noescape
func sock_open(af int32, socktype int32, proto int32, fd unsafe.Pointer) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_bind
//go:noescape
func sock_bind(fd int32, addr unsafe.Pointer, addrlen uintptr) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_connect
//go:noescape
func sock_connect(fd int32, addr unsafe.Pointer, addrlen uintptr, port uint32) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_listen
//go:noescape
func sock_listen(fd int32, backlog int32) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_getsockopt
//go:noescape
func sock_getsockopt(fd int32, level uint32, name uint32, value unsafe.Pointer, valueLen uint32) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_setsockopt
//go:noescape
func sock_setsockopt(fd int32, level uint32, name uint32, value unsafe.Pointer, valueLen uint32) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_getlocaladdr
//go:noescape
func sock_getlocaladdr(fd int32, addr unsafe.Pointer) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_getpeeraddr
//go:noescape
func sock_getpeeraddr(fd int32, addr unsafe.Pointer) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_recv_from
//go:noescape
func sock_recv_from(
	fd int32,
	iovs unsafe.Pointer,
	iovsCount int32,
	addr unsafe.Pointer,
	iflags int32,
	port unsafe.Pointer,
	nread unsafe.Pointer,
	oflags unsafe.Pointer,
) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_send_to
//go:noescape
func sock_send_to(
	fd int32,
	iovs unsafe.Pointer,
	iovsCount int32,
	addr unsafe.Pointer,
	port int32,
	flags int32,
	nwritten unsafe.Pointer,
) syscall.Errno

// //go:wasmimport wasi_snapshot_preview1 sock_getaddrinfo
// //go:noescape
// func sock_getaddrinfo(
// 	node unsafe.Pointer,
// 	nodeLen uint32,
// 	service unsafe.Pointer,
// 	serviceLen uint32,
// 	hints unsafe.Pointer,
// 	res unsafe.Pointer,
// 	maxResLen uint32,
// 	resLen unsafe.Pointer,
// ) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_shutdown
func sock_shutdown(fd, how int32) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_getaddrport
//go:noescape
func sock_getaddrip(
	networkptr unsafe.Pointer, networklen uint32,
	addressptr unsafe.Pointer, addresslen uint32,
	ipres unsafe.Pointer, maxResLen uint32, ipreslen unsafe.Pointer,
) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_getaddrport
//go:noescape
func sock_getaddrport(
	networkptr unsafe.Pointer, networklen uint32,
	serviceptr unsafe.Pointer, servicelen uint32,
	portptr unsafe.Pointer,
) syscall.Errno
