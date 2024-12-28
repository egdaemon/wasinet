//go:build wasip1

package wasip1net

import (
	"net"
	"os"
	"runtime"
	"syscall"
	"time"
	_ "unsafe"
)

// https://github.com/WebAssembly/WASI/blob/a2b96e81c0586125cc4dc79a5be0b78d9a059925/legacy/preview1/docs.md#filetype

type filetype uint8

const (
	ftype_unknown          filetype = iota // The type of the file descriptor or file is unknown or is different from any of the other types specified.
	ftype_block_device                     // The file descriptor or file refers to a block device inode.
	ftype_character_device                 // The file descriptor or file refers to a character device inode.
	ftype_directory                        // The file descriptor or file refers to a directory inode.
	ftype_regular_file                     // The file descriptor or file refers to a regular file inode.
	ftype_socket_dgram                     // The file descriptor or file refers to a datagram socket.
	ftype_socket_stream                    // The file descriptor or file refers to a byte-stream socket.
	ftype_symbolic_link                    // The file refers to a symbolic link inode.
)

const (
	readSyscallName  = "fd_read"
	writeSyscallName = "fd_write"
)

type pollfd interface {
	Accept() (int, syscall.Sockaddr, string, error)
	Close() error
	Dup() (int, string, error)
	Fchdir() error
	Fchmod(mode uint32) error
	Fchown(uid int, gid int) error
	Fstat(s *syscall.Stat_t) error
	Fsync() error
	Ftruncate(size int64) error
	Init(net string, pollable bool) error
	Pread(p []byte, off int64) (int, error)
	Pwrite(p []byte, off int64) (int, error)
	RawControl(f func(uintptr)) error
	RawRead(f func(uintptr) bool) error
	RawWrite(f func(uintptr) bool) error
	Read(p []byte) (int, error)
	ReadDir(buf []byte, cookie uint64) (int, error)
	ReadDirent(buf []byte) (int, error)
	ReadFrom(p []byte) (int, syscall.Sockaddr, error)
	ReadFromInet4(p []byte, from *syscall.SockaddrInet4) (int, error)
	ReadFromInet6(p []byte, from *syscall.SockaddrInet6) (int, error)
	ReadMsg(p []byte, oob []byte, flags int) (int, int, int, syscall.Sockaddr, error)
	ReadMsgInet4(p []byte, oob []byte, flags int, sa4 *syscall.SockaddrInet4) (int, int, int, error)
	ReadMsgInet6(p []byte, oob []byte, flags int, sa6 *syscall.SockaddrInet6) (int, int, int, error)
	Seek(offset int64, whence int) (int64, error)
	SetBlocking() error
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	Shutdown(how int) error
	WaitWrite() error
	Write(p []byte) (int, error)
	WriteMsg(p []byte, oob []byte, sa syscall.Sockaddr) (int, int, error)
	WriteMsgInet4(p []byte, oob []byte, sa *syscall.SockaddrInet4) (int, int, error)
	WriteMsgInet6(p []byte, oob []byte, sa *syscall.SockaddrInet6) (int, int, error)
	WriteOnce(p []byte) (int, error)
	WriteTo(p []byte, sa syscall.Sockaddr) (int, error)
	WriteToInet4(p []byte, sa *syscall.SockaddrInet4) (int, error)
	WriteToInet6(p []byte, sa *syscall.SockaddrInet6) (int, error)
}

type unknownAddr struct{}

func (unknownAddr) Network() string { return "unknown" }
func (unknownAddr) String() string  { return "unknown" }

// wrapSyscallError takes an error and a syscall name. If the error is
// a syscall.Errno, it wraps it in an os.SyscallError using the syscall name.
func wrapSyscallError(name string, err error) error {
	if _, ok := err.(syscall.Errno); ok {
		err = os.NewSyscallError(name, err)
	}
	return err
}

// This helper is implemented in the syscall package. It means we don't have
// to redefine the fd_fdstat_get host import or the fdstat struct it
// populates.
//
// func fd_fdstat_get_type(fd int) (uint8, error)
//
// go:linkname fd_fdstat_get_type syscall.fd_fdstat_get_type
func net_fd_fdstat_get_type(fd int) (uint8, error) {
	// res, err := fd_fdstat_get_type(fd)
	// log.Println("fdstat_get_type", fd, res, err)
	return uint8(ftype_socket_stream), nil
}

func fileConnNet(filetype syscall.Filetype) (string, error) {
	switch filetype {
	case syscall.FILETYPE_SOCKET_STREAM:
		return "tcp", nil
	case syscall.FILETYPE_SOCKET_DGRAM:
		return "udp", nil
	default:
		return "", syscall.ENOTSOCK
	}
}

//go:linkname newFile net.newUnixFile
func newFile(fd int, name string) *os.File

func newFD(f *os.File) (*netFD, error) {
	filetype, err := net_fd_fdstat_get_type(f.PollFD().Sysfd)
	if err != nil {
		return nil, err
	}
	net, err := fileConnNet(filetype)
	if err != nil {
		return nil, err
	}
	pfd := f.PollFD().Copy()
	fd := newPollFD(net, &pfd)
	if err := fd.init(); err != nil {
		pfd.Close()
		return nil, err
	}

	return fd, nil
}

func newConn(f *os.File) (net.Conn, error) {
	fd, err := newFD(f)
	if err != nil {
		return nil, err
	}
	return newFileConn(fd), nil
}

func newFileConn(fd *netFD) net.Conn {
	switch fd.net {
	case "tcp":
		return &TCPConn{conn{fd: fd}}
	// case "udp":
	// 	return &UDPConn{conn{fd: fd}}
	default:
		panic("unsupported network for file connection: " + fd.net)
	}
}

func PacketConn(fd uintptr, family int, sotype int, netw string, laddr, raddr net.Addr) (net.PacketConn, error) {
	pfd, err := newFD(Socket(fd))
	if err != nil {
		return nil, err
	}
	return makePacketConn(&packetConn{conn: &conn{pfd}}), nil
}

func setReadBuffer(fd *netFD, bytes int) (err error) {
	// err := fd.pfd.SetsockoptInt(syscall.SOL_SOCKET, syscall.SO_RCVBUF, bytes)
	runtime.KeepAlive(fd)
	return wrapSyscallError("setsockopt", err)
}

func setWriteBuffer(fd *netFD, bytes int) (err error) {
	// err := fd.pfd.SetsockoptInt(syscall.SOL_SOCKET, syscall.SO_SNDBUF, bytes)
	runtime.KeepAlive(fd)
	return wrapSyscallError("setsockopt", err)
}

func setKeepAlive(fd *netFD, keepalive bool) (err error) {
	// err := fd.pfd.SetsockoptInt(syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, boolint(keepalive))
	runtime.KeepAlive(fd)
	return wrapSyscallError("setsockopt", err)
}

func setLinger(fd *netFD, sec int) (err error) {
	// var l syscall.Linger
	// if sec >= 0 {
	// 	l.Onoff = 1
	// 	l.Linger = int32(sec)
	// } else {
	// 	l.Onoff = 0
	// 	l.Linger = 0
	// }
	// err := fd.pfd.SetsockoptLinger(syscall.SOL_SOCKET, syscall.SO_LINGER, &l)
	runtime.KeepAlive(fd)
	return wrapSyscallError("setsockopt", err)
}
