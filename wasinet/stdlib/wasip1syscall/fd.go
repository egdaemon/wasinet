package wasip1syscall

import (
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/egdaemon/wasinet/wasinet/ffi"
	"github.com/egdaemon/wasinet/wasinet/ffierrors"
)

func RecvFromsingle(fd int, b []byte, flags int32) (n int, addr RawSocketAddress, oflags int32, err error) {
	return recvfrom(fd, [][]byte{b}, flags)
}

func recvfrom(fd int, iovs [][]byte, flags int32) (n int, addr RawSocketAddress, oflags int32, err error) {
	vecs := ffi.VectorSlice(iovs...)
	iovsptr, iovslen := ffi.Slice(vecs)
	addrptr, addrlen := ffi.Pointer(&addr)

	errno := sock_recv_from(
		int32(fd),
		iovsptr, iovslen,
		addrptr, addrlen,
		flags,
		unsafe.Pointer(&n),
		unsafe.Pointer(&oflags),
	)

	runtime.KeepAlive(addrptr)
	runtime.KeepAlive(iovsptr)
	runtime.KeepAlive(iovs)
	return n, addr, oflags, ffierrors.Error(errno)
}

func SendToSingle(fd int, b []byte, addr RawSocketAddress, flags int32) (int, error) {
	return sendto(fd, [][]byte{b}, addr, flags)
}

func sendto(fd int, iovs [][]byte, addr RawSocketAddress, flags int32) (int, error) {
	vecs := ffi.VectorSlice(iovs...)
	iovsptr, iovslen := ffi.Slice(vecs)
	addrptr, addrlen := ffi.Pointer(&addr)

	nwritten := int(0)
	errno := sock_send_to(
		int32(fd),
		iovsptr, iovslen,
		addrptr, addrlen,
		flags,
		unsafe.Pointer(&nwritten),
	)
	runtime.KeepAlive(addr)
	runtime.KeepAlive(iovs)
	return nwritten, ffierrors.Error(errno)
}

func getrawsockname(fd int) (rsa RawSocketAddress, err error) {
	rsaptr, rsalength := ffi.Pointer(&rsa)
	errno := ffierrors.Error(sock_getlocaladdr(int32(fd), rsaptr, rsalength))
	return rsa, errno
}

func Getsockname(fd int) (sa sockaddr, err error) {
	rsa, err := getrawsockname(fd)
	if err != nil {
		return nil, err
	}
	return rawtosockaddr(&rsa)
}

func getrawpeername(fd int) (rsa RawSocketAddress, err error) {
	rsaptr, rsalength := ffi.Pointer(&rsa)
	errno := sock_getpeeraddr(int32(fd), rsaptr, rsalength)
	return rsa, ffierrors.Error(errno)
}

func Getpeername(fd int) (sockaddr, error) {
	rsa, err := getrawpeername(fd)
	if err != nil {
		return nil, err
	}
	return rawtosockaddr(&rsa)
}

func SetsockoptTimeval(fd int, level uint32, opt uint32, d time.Duration) syscall.Errno {
	type Timeval struct {
		Sec  int64
		Usec int64
	}

	secs := d.Truncate(time.Second)
	milli := d - secs
	tval := &Timeval{Sec: int64(secs / time.Second), Usec: milli.Milliseconds()}
	tvalptr, tvallen := ffi.Pointer(tval)
	return sock_setsockopt(int32(fd), level, opt, tvalptr, tvallen)
}

func Connect(fd int, rsa *RawSocketAddress) error {
	rawaddr, rawaddrlen := ffi.Pointer(rsa)
	err := ffierrors.Error(sock_connect(int32(fd), rawaddr, rawaddrlen))
	runtime.KeepAlive(rsa)
	return err
}

func SetSockoptInt(fd, level, opt int, value int) error {
	var n = int32(value)
	errno := ffierrors.Error(sock_setsockopt(int32(fd), uint32(level), uint32(opt), unsafe.Pointer(&n), 4))
	return errno
}

func GetSockoptInt(fd, level, opt int) (value int, err error) {
	var n int32
	errno := ffierrors.Error(sock_getsockopt(int32(fd), uint32(level), uint32(opt), unsafe.Pointer(&n), 4))
	return int(n), errno
}

func Bind(fd int, rsa *RawSocketAddress) error {
	rawaddr, rawaddrlen := ffi.Pointer(&rsa)
	errno := ffierrors.Error(sock_bind(int32(fd), rawaddr, rawaddrlen))
	runtime.KeepAlive(rsa)
	return errno
}

func Socket(af, sotype, proto int) (fd int, err error) {
	var newfd int32 = -1
	errno := ffierrors.Error(sock_open(int32(af), int32(sotype), int32(proto), unsafe.Pointer(&newfd)))
	return int(newfd), errno
}

func Listen(fd int, backlog int) error {
	return ffierrors.Error(sock_listen(int32(fd), int32(backlog)))
}
