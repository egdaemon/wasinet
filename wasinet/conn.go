package wasinet

import (
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/egdaemon/wasinet/wasinet/internal/errorsx"
	"github.com/egdaemon/wasinet/wasinet/stdlib/wasip1net"
	"github.com/egdaemon/wasinet/wasinet/stdlib/wasip1syscall"
)

type sockaddr interface {
	Sockaddr() wasip1syscall.RawSocketAddress
}

// In Go 1.21, the net package cannot initialize the local and remote addresses
// of network connections. For this reason, we use this function to retreive the
// addresses and return a wrapped net.Conn with LocalAddr/RemoteAddr implemented.
func initconn(c net.Conn) (x net.Conn, err error) {
	syscallConn, ok := c.(syscall.Conn)
	if !ok {
		return c, nil
	}

	defer func() {
		if err == nil {
			return
		}

		if c != nil {
			c.Close()
		}
	}()

	rawConn, err := syscallConn.SyscallConn()
	if err != nil {
		return nil, fmt.Errorf("syscall.Conn.SyscallConn: %w", err)
	}
	cerr := rawConn.Control(func(fd uintptr) {
		var (
			addr sockaddr
			peer sockaddr
		)

		if addr, err = wasip1syscall.Getsockname(int(fd)); err != nil {
			err = os.NewSyscallError("getsockname", err)
			return
		}

		if peer, err = wasip1syscall.Getpeername(int(fd)); err != nil {
			err = os.NewSyscallError("getpeername", err)
			return
		}

		wasip1net.SetNetAddr(syscall.SOCK_STREAM, c.LocalAddr(), addr)
		wasip1net.SetNetAddr(syscall.SOCK_STREAM, c.RemoteAddr(), peer)

		// if _, unix := addr.(*sockaddrUnix); unix {
		// 	c = &unixConn{Conn: c}
		// }
	})

	return c, errorsx.Compact(err, cerr)
}
