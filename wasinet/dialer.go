package wasinet

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net"
	"os"
	"syscall"
	"time"

	"github.com/egdaemon/wasinet/wasinet/ffierrors"
	"github.com/egdaemon/wasinet/wasinet/internal/errorsx"
	"github.com/egdaemon/wasinet/wasinet/stdlib/wasip1net"
	"github.com/egdaemon/wasinet/wasinet/stdlib/wasip1syscall"
)

// Dialer is a type similar to net.Dialer but it uses the dial functions defined
// in this package instead of those from the standard library.
//
// For details about the configuration, see: https://pkg.go.dev/net#Dialer
//
// Note that depending on the WebAssembly runtime being employed, certain
// functionalities of the Dialer may not be available.
type Dialer struct {
	Timeout        time.Duration
	Deadline       time.Time
	LocalAddr      net.Addr
	DualStack      bool
	FallbackDelay  time.Duration
	Resolver       *net.Resolver   // ignored
	Cancel         <-chan struct{} // ignored
	Control        func(network, address string, c syscall.RawConn) error
	ControlContext func(ctx context.Context, network, address string, c syscall.RawConn) error
}

func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	timeout := d.Timeout
	if !d.Deadline.IsZero() {
		dl := max(0, time.Until(d.Deadline))
		timeout = min(max(d.Timeout, dl), dl)
	}

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	if d.LocalAddr != nil {
		slog.WarnContext(ctx, "wasip1.Dialer: LocalAddr not yet supported on GOOS=wasip1")
	}
	if d.Resolver != nil {
		slog.WarnContext(ctx, "wasip1.Dialer: Resolver ignored because it is not supported on GOOS=wasip1")
	}
	if d.Cancel != nil {
		slog.WarnContext(ctx, "wasip1.Dialer: Cancel channel not implemented on GOOS=wasip1")
	}
	if d.Control != nil {
		slog.WarnContext(ctx, "wasip1.Dialer: Control function not yet supported on GOOS=wasip1")
	}
	if d.ControlContext != nil {
		slog.WarnContext(ctx, "wasip1.Dialer: ControlContext function not yet supported on GOOS=wasip1")
	}
	// TOOD:
	// - use LocalAddr to bind to a socket prior to establishing the connection
	// - use DualStack and FallbackDelay
	// - use Control and ControlContext functions
	// - emulate the Cancel channel with context.Context
	return DialContext(ctx, network, address)
}

// Dial connects to the address on the named network.
func Dial(network, address string) (net.Conn, error) {
	return DialContext(context.Background(), network, address)
}

// DialContext is a variant of Dial that accepts a context.
func DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	addrs, err := wasip1syscall.LookupAddress(ctx, opdial, network, address)
	if err != nil {
		return nil, netOpErr(opdial, unresolvedaddr(network, address), err)
	}

	for _, addr := range addrs {
		var conn net.Conn
		conn, err = dialAddr(ctx, addr)
		if err == nil {
			return conn, nil
		}

		if ctx.Err() != nil {
			break
		}
	}

	return nil, netOpErr(opdial, unresolvedaddr(network, address), err)
}

func dialAddr(ctx context.Context, addr net.Addr) (net.Conn, error) {
	af := wasip1syscall.NetaddrAFFamily(addr)
	sotype, err := socketType(addr)
	if err != nil {
		return nil, os.NewSyscallError("socket", err)
	}

	fd, err := wasip1syscall.Socket(af, sotype, netaddrproto(addr))
	if err != nil {
		return nil, os.NewSyscallError("socket", err)
	}
	defer func() {
		if fd >= 0 {
			syscall.Close(fd)
		}
	}()

	if sotype == syscall.SOCK_DGRAM && af != syscall.AF_UNIX {
		if err := wasip1syscall.SetSockoptInt(fd, SOL_SOCKET, SO_BROADCAST, 1); err != nil {
			// If the system does not support broadcast we should still be able
			// to use the datagram socket.
			switch {
			case errors.Is(err, syscall.EINVAL):
			case errors.Is(err, syscall.ENOPROTOOPT):
			default:
				return nil, os.NewSyscallError("setsockopt", err)
			}
		}
	}

	caddr, err := wasip1syscall.NetaddrToRaw(addr)
	if err != nil {
		return nil, os.NewSyscallError("sockaddr", err)
	}

	var inProgress bool
	switch err := wasip1syscall.Connect(fd, caddr); err {
	case nil:
	case ffierrors.EINPROGRESS:
		inProgress = true
	default:
		return nil, os.NewSyscallError("connect", err)
	}

	if sotype == syscall.SOCK_DGRAM {
		localaddr, err := wasip1syscall.Getsockname(fd)
		if err != nil {
			return nil, err
		}

		peeraddr, err := wasip1syscall.Getpeername(fd)
		if err != nil {
			return nil, err
		}

		laddr := wasip1syscall.SocketAddressFormat(af, sotype)(localaddr)
		raddr := wasip1syscall.SocketAddressFormat(af, sotype)(peeraddr)
		_, _ = laddr, raddr
		return wasip1net.Conn(wasip1net.Socket(uintptr(fd)))
	}

	sconn := wasip1net.Socket(uintptr(fd))
	fd = -1 // now the *netFD owns the file descriptor
	defer func() {
		if err == nil {
			return
		}

		err = sconn.Close()
	}()
	if inProgress {
		rawConn, err := sconn.SyscallConn()
		if err != nil {
			return nil, err
		}
		errch := make(chan error)
		go func() {
			var err error
			cerr := rawConn.Write(func(fd uintptr) bool {
				var value int
				value, err = wasip1syscall.GetSockoptInt(int(fd), SOL_SOCKET, syscall.SO_ERROR)
				if err != nil {
					return true // done
				}
				switch syscall.Errno(value) {
				case syscall.EINPROGRESS, syscall.EINTR:
					return false // continue
				case syscall.EISCONN:
					err = nil
					return true
				case syscall.Errno(0):
					// The net poller can wake up spuriously. Check that we are
					// are really connected.
					_, err := wasip1syscall.Getpeername(int(fd))
					return err == nil
				default:
					err = syscall.Errno(value)
					return true
				}
			})
			errch <- errorsx.Compact(err, cerr)
		}()

		select {
		case err := <-errch:
			if err != nil {
				return nil, os.NewSyscallError("connect", err)
			}
		case <-ctx.Done():
			// This should interrupt the async connect operation handled by the
			// goroutine.
			sconn.Close()
			// Wait for the goroutine to complete, we can safely discard the
			// error here because we don't care about the socket anymore.
			<-errch
			return nil, context.Cause(ctx)
		}
	}

	slog.Debug("------------ critical area initiated ------------")
	defer slog.Debug("------------ critical area completed ------------")

	// switch fd.laddr.(type) {
	// case *TCPAddr:
	// 	return newTCPConn(fd, defaultTCPKeepAliveIdle, KeepAliveConfig{}, testPreHookSetKeepAlive, testHookSetKeepAlive), nil
	// case *UDPAddr:
	// 	return newUDPConn(fd), nil
	// case *IPAddr:
	// 	return newIPConn(fd), nil
	// case *UnixAddr:
	// 	return newUnixConn(fd), nil
	// }go:

	log.Println("HELLO WORLD", sconn.Fd())
	return initconn(errorsx.Must(wasip1net.Conn(sconn)))
	// return net.RawFileConn(sconn.fd)
	// return makeConn(sconn)

}
