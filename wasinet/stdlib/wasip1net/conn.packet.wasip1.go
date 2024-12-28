//go:build wasip1

package wasip1net

import (
	"io"
	"log"
	"net"
	"net/netip"
	"syscall"
	"time"

	"github.com/egdaemon/wasinet/wasinet/internal/errorsx"
	"github.com/egdaemon/wasinet/wasinet/stdlib/wasip1syscall"
)

type packetConn struct {
	conn *conn
}

func (c *packetConn) Close() error {
	return c.conn.Close()
}

func (c *packetConn) CloseRead() (err error) {
	return c.conn.fd.closeRead()
}

func (c *packetConn) CloseWrite() (err error) {
	return c.conn.fd.closeWrite()
}

func (c *packetConn) Read(b []byte) (int, error) {
	n, _, err := c.ReadFrom(b)
	return n, err
}

func (c *packetConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	switch c.conn.LocalAddr().(type) {
	case *net.UDPAddr:
		n, _, _, addr, err = c.ReadMsgUDP(b, nil)
	default:
		n, _, _, addr, err = c.ReadMsgUnix(b, nil)
	}
	return
}

func (c *packetConn) ReadMsgUnix(b, oob []byte) (n, oobn, flags int, addr *net.UnixAddr, err error) {
	rawConnErr := c.conn.SyscallConn().Read(func(fd uintptr) (done bool) {
		var raw wasip1syscall.RawSocketAddress
		var oflags int32
		n, raw, oflags, err = wasip1syscall.RecvFromsingle(int(fd), b, 0)
		if err == syscall.EAGAIN {
			return false
		}
		if err == syscall.EINVAL {
			// This error occurs when the socket is shutdown asynchronusly
			// by a call to CloseRead.
			n, err = 0, io.EOF
		} else {
			if addr, err = wasip1syscall.NetUnix(raw); err != nil {
				return true
			}

		}
		flags = int(oflags)
		return true
	})

	if rawConnErr != nil {
		err = rawConnErr
	}
	if n == 0 && err == nil {
		err = io.EOF
	}
	return
}

func (c *packetConn) ReadMsgUDP(b, oob []byte) (n, oobn, flags int, addr *net.UDPAddr, err error) {
	n, oobn, flags, addrPort, err := c.ReadMsgUDPAddrPort(b, oob)
	return n, oobn, flags, net.UDPAddrFromAddrPort(addrPort), err
}

func (c *packetConn) ReadMsgUDPAddrPort(b, oob []byte) (n, oobn, flags int, addrPort netip.AddrPort, err error) {
	werr := c.conn.SyscallConn().Read(func(fd uintptr) (done bool) {
		var remote wasip1syscall.RawSocketAddress
		var oflags int32
		n, remote, oflags, err = wasip1syscall.RecvFromsingle(int(fd), b, 0)
		if err == syscall.EAGAIN {
			return false
		}
		if err == syscall.EINVAL {
			// This error occurs when the socket is shutdown asynchronusly
			// by a call to CloseRead.
			n, err = 0, io.EOF
			return true
		}
		if err != nil {
			return true
		}

		if addrPort, err = wasip1syscall.Netipaddrport(remote); err != nil {
			log.Println("failed to decode address", remote.Family, syscall.AF_INET, syscall.AF_INET6, err)
			return true
		}

		flags = int(oflags)
		return true
	})

	if n == 0 && werr == nil {
		err = io.EOF
	}

	return n, oobn, flags, addrPort, err
}

func (c *packetConn) Write(b []byte) (int, error) {
	return c.WriteTo(b, c.conn.RemoteAddr())
}

func (c *packetConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	switch a := addr.(type) {
	case *net.UDPAddr:
		if _, ok := c.conn.LocalAddr().(*net.UDPAddr); ok {
			n, _, err := c.WriteMsgUDP(b, nil, a)
			return n, err
		}
	case *net.UnixAddr:
		if _, ok := c.conn.LocalAddr().(*net.UnixAddr); ok {
			n, _, err := c.WriteMsgUnix(b, nil, a)
			return n, err
		}
	}
	return 0, &net.OpError{
		Op:     "write",
		Net:    c.conn.LocalAddr().Network(),
		Addr:   c.conn.LocalAddr(),
		Source: addr,
		Err:    net.InvalidAddrError("address type mismatch"),
	}
}

func (c *packetConn) WriteMsgUnix(b, oob []byte, addr *net.UnixAddr) (n, oobn int, err error) {
	werr := c.conn.SyscallConn().Write(func(fd uintptr) (done bool) {
		n, err = wasip1syscall.SendToSingle(int(fd), b, wasip1syscall.NetUnixToRaw(addr), 0)
		return err != syscall.EAGAIN
	})
	return n, oobn, errorsx.Compact(werr, err)
}

func (c *packetConn) WriteMsgUDP(b, oob []byte, addr *net.UDPAddr) (n, oobn int, err error) {
	return c.WriteMsgUDPAddrPort(b, oob, addr.AddrPort())
}

func (c *packetConn) WriteMsgUDPAddrPort(b, oob []byte, addrPort netip.AddrPort) (n, oobn int, err error) {
	cerr := c.conn.SyscallConn().Write(func(fd uintptr) (done bool) {
		n, err = wasip1syscall.SendToSingle(int(fd), b, wasip1syscall.NetipAddrPortToRaw(addrPort), 0)
		return err != syscall.EAGAIN
	})
	return n, oobn, errorsx.Compact(cerr, err)
}

func (c *packetConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *packetConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *packetConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *packetConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *packetConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
