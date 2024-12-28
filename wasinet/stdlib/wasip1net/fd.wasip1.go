//go:build wasip1

package wasip1net

import (
	"net"
	"os"
	"runtime"
	"syscall"
	"time"
)

// Network file descriptor.
type netFD struct {
	pfd pollfd

	// immutable until Close
	family      int
	sotype      int
	isConnected bool // handshake completed or use of association with peer
	net         string
	laddr       net.Addr
	raddr       net.Addr

	// The only networking available in WASI preview 1 is the ability to
	// sock_accept on a pre-opened socket, and then fd_read, fd_write,
	// fd_close, and sock_shutdown on the resulting connection. We
	// intercept applicable netFD calls on this instance, and then pass
	// the remainder of the netFD calls to fakeNetFD.
	// *fakeNetFD
}

func newPollFD(network string, pfd pollfd) *netFD {
	var laddr net.Addr
	var raddr net.Addr
	// WASI preview 1 does not have functions like getsockname/getpeername,
	// so we cannot get access to the underlying IP address used by connections.
	//
	// However, listeners created by FileListener are of type *TCPListener,
	// which can be asserted by a Go program. The (*TCPListener).Addr method
	// documents that the returned value will be of type *TCPAddr, we satisfy
	// the documented behavior by creating addresses of the expected type here.
	switch network {
	case "tcp":
		laddr = new(net.TCPAddr)
		raddr = new(net.TCPAddr)
	case "udp":
		laddr = new(net.UDPAddr)
		raddr = new(net.UDPAddr)
	default:
		laddr = unknownAddr{}
		raddr = unknownAddr{}
	}

	return &netFD{
		pfd:   pfd,
		net:   network,
		laddr: laddr,
		raddr: raddr,
	}
}

func (fd *netFD) init() error {
	return fd.pfd.Init(fd.net, true)
}

func (fd *netFD) name() string {
	return "unknown"
}

func (fd *netFD) accept() (netfd *netFD, err error) {
	// if fd.fakeNetFD != nil {
	// 	return fd.fakeNetFD.accept(fd.laddr)
	// }
	d, _, errcall, err := fd.pfd.Accept()
	if err != nil {
		if errcall != "" {
			err = wrapSyscallError(errcall, err)
		}
		return nil, err
	}

	netfd = newPollFD("tcp", newFile(d, "").PollFD())
	if err = netfd.init(); err != nil {
		netfd.Close()
		return nil, err
	}
	return netfd, nil
}

func (fd *netFD) setAddr(laddr, raddr net.Addr) {
	fd.laddr = laddr
	fd.raddr = raddr
	runtime.SetFinalizer(fd, (*netFD).Close)
}

func (fd *netFD) Close() error {
	// if fd.fakeNetFD != nil {
	// 	return fd.fakeNetFD.Close()
	// }
	runtime.SetFinalizer(fd, nil)
	return fd.pfd.Close()
}

func (fd *netFD) shutdown(how int) error {
	// if fd.fakeNetFD != nil {
	// 	return nil
	// }
	err := fd.pfd.Shutdown(how)
	runtime.KeepAlive(fd)
	return wrapSyscallError("shutdown", err)
}

func (fd *netFD) closeRead() error {
	return fd.shutdown(syscall.SHUT_RD)
}

func (fd *netFD) closeWrite() error {
	return fd.shutdown(syscall.SHUT_WR)
}

func (fd *netFD) Read(p []byte) (n int, err error) {
	// if fd.fakeNetFD != nil {
	// 	return fd.fakeNetFD.Read(p)
	// }
	n, err = fd.pfd.Read(p)
	runtime.KeepAlive(fd)
	return n, wrapSyscallError(readSyscallName, err)
}

func (fd *netFD) Write(p []byte) (nn int, err error) {
	// if fd.fakeNetFD != nil {
	// 	return fd.fakeNetFD.Write(p)
	// }
	nn, err = fd.pfd.Write(p)
	runtime.KeepAlive(fd)
	return nn, wrapSyscallError(writeSyscallName, err)
}

func (fd *netFD) SetDeadline(t time.Time) error {
	// if fd.fakeNetFD != nil {
	// 	return fd.fakeNetFD.SetDeadline(t)
	// }
	return fd.pfd.SetDeadline(t)
}

func (fd *netFD) SetReadDeadline(t time.Time) error {
	// if fd.fakeNetFD != nil {
	// 	return fd.fakeNetFD.SetReadDeadline(t)
	// }
	return fd.pfd.SetReadDeadline(t)
}

func (fd *netFD) SetWriteDeadline(t time.Time) error {
	// if fd.fakeNetFD != nil {
	// 	return fd.fakeNetFD.SetWriteDeadline(t)
	// }
	return fd.pfd.SetWriteDeadline(t)
}

func (fd *netFD) dup() (f *os.File, err error) {
	ns, call, err := fd.pfd.Dup()
	if err != nil {
		if call != "" {
			err = os.NewSyscallError(call, err)
		}
		return nil, err
	}

	return newFile(ns, fd.name()), nil
}
