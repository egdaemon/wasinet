//go:build !wasip1 && !windows

package wnetruntime

import (
	"context"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/egdaemon/wasinet/wasinet/ffi"
	"github.com/egdaemon/wasinet/wasinet/stdlib/wasip1syscall"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestClassifyLookupErrno(t *testing.T) {
	t.Run("syscall errno passes through unchanged", func(t *testing.T) {
		require.Equal(t, syscall.ECONNREFUSED, classifyLookupErrno(syscall.ECONNREFUSED))
	})

	t.Run("context canceled maps to ECANCELED", func(t *testing.T) {
		require.Equal(t, syscall.ECANCELED, classifyLookupErrno(context.Canceled))
	})

	t.Run("context deadline exceeded maps to ETIMEDOUT", func(t *testing.T) {
		require.Equal(t, syscall.ETIMEDOUT, classifyLookupErrno(context.DeadlineExceeded))
	})

	t.Run("timeout error maps to ETIMEDOUT", func(t *testing.T) {
		err := &net.DNSError{Err: "i/o timeout", Name: "example.invalid", IsTimeout: true}
		require.Equal(t, syscall.ETIMEDOUT, classifyLookupErrno(err))
	})

	t.Run("no such host maps to ENOENT rather than panicking", func(t *testing.T) {
		err := &net.DNSError{Err: "no such host", Name: "example.invalid", IsNotFound: true}
		require.NotPanics(t, func() {
			require.Equal(t, syscall.ENOENT, classifyLookupErrno(err))
		})
	})

	t.Run("arbitrary error maps to ENOENT rather than panicking", func(t *testing.T) {
		require.NotPanics(t, func() {
			require.Equal(t, syscall.ENOENT, classifyLookupErrno(fmt.Errorf("some unclassified failure")))
		})
	})
}

func TestSocketAddrIP(t *testing.T) {
	call := func(t *testing.T, fn AddrIPHostFn) syscall.Errno {
		t.Helper()

		networkptr, networklen := ffi.String("tcp")
		addressptr, addresslen := ffi.String("example.invalid")
		ipres := make([]byte, net.IPv6len)
		ipresptr, maxResLen := ffi.Slice(ipres)
		var ipreslen uint32
		ipreslenptr, _ := ffi.Pointer(&ipreslen)

		return fn(
			context.Background(), ffi.Native{},
			uintptr(networkptr), networklen,
			uintptr(addressptr), addresslen,
			uintptr(ipresptr), maxResLen,
			uintptr(ipreslenptr),
		)
	}

	t.Run("lookup failure is translated to wasi errno numbering, not the raw host errno", func(t *testing.T) {
		fn := SocketAddrIP(func(ctx context.Context, network, address string) ([]net.IP, error) {
			return nil, &net.DNSError{Err: "no such host", Name: address, IsNotFound: true}
		})

		errno := call(t, fn)

		require.Equal(t, wasip1syscall.ENOENT, errno)
		require.NotEqual(t, syscall.Errno(22), errno, "must not leak the raw linux errno (misread as EFBIG under wasi numbering) across the ffi boundary")
	})

	t.Run("timeout is preserved distinctly from a not-found lookup", func(t *testing.T) {
		fn := SocketAddrIP(func(ctx context.Context, network, address string) ([]net.IP, error) {
			return nil, &net.DNSError{Err: "i/o timeout", Name: address, IsTimeout: true}
		})

		errno := call(t, fn)

		require.Equal(t, wasip1syscall.ETIMEDOUT, errno)
	})

	t.Run("success writes resolved addresses and length", func(t *testing.T) {
		want := net.ParseIP("93.184.216.34")
		fn := SocketAddrIP(func(ctx context.Context, network, address string) ([]net.IP, error) {
			return []net.IP{want}, nil
		})

		networkptr, networklen := ffi.String("tcp")
		addressptr, addresslen := ffi.String("example.com")
		ipres := make([]byte, net.IPv6len)
		ipresptr, maxResLen := ffi.Slice(ipres)
		var ipreslen uint32
		ipreslenptr, _ := ffi.Pointer(&ipreslen)

		errno := fn(
			context.Background(), ffi.Native{},
			uintptr(networkptr), networklen,
			uintptr(addressptr), addresslen,
			uintptr(ipresptr), maxResLen,
			uintptr(ipreslenptr),
		)

		require.Equal(t, syscall.Errno(0), errno)
		require.Equal(t, uint32(net.IPv6len), ipreslen)
		require.Equal(t, []byte(want.To16()), ipres)
	})
}

func TestSocketShutdown(t *testing.T) {
	// the guest speaks wasi numbering, sdflags: rd = 1, wr = 2, rdwr = 3.
	// the host fn receives the host's numbering, which differs (linux rdwr = 2).
	const (
		wasiRd   = 0x1
		wasiWr   = 0x2
		wasiRdWr = wasiRd | wasiWr
	)

	call := func(how int32) (host int, called bool, errno syscall.Errno) {
		fn := SocketShutdown(func(ctx context.Context, fd, how int) error {
			host, called = how, true
			return nil
		})

		errno = fn(context.Background(), ffi.Native{}, 3, how)
		return host, called, errno
	}

	t.Run("wasi read maps to the host read", func(t *testing.T) {
		host, called, errno := call(wasiRd)
		require.Equal(t, syscall.Errno(0), errno)
		require.True(t, called)
		require.Equal(t, unix.SHUT_RD, host)
	})

	t.Run("wasi write maps to the host write", func(t *testing.T) {
		host, called, errno := call(wasiWr)
		require.Equal(t, syscall.Errno(0), errno)
		require.True(t, called)
		require.Equal(t, unix.SHUT_WR, host)
	})

	t.Run("wasi read and write maps to the host read and write, not an invalid value", func(t *testing.T) {
		host, called, errno := call(wasiRdWr)
		require.Equal(t, syscall.Errno(0), errno)
		require.True(t, called)
		require.Equal(t, unix.SHUT_RDWR, host)
	})

	t.Run("unknown flags are rejected without reaching the host", func(t *testing.T) {
		for _, how := range []int32{0, 4, 7, -1} {
			_, called, errno := call(how)
			require.False(t, called, "how=%d", how)
			require.Equal(t, wasip1syscall.EINVAL, errno, "how=%d", how)
		}
	})
}

// a real connection must observe the peer's shutdown, which only happens when the
// guest's flags are translated properly.
func TestNetworkShutdownReleasesPeer(t *testing.T) {
	l, err := net.Listen("unix", filepath.Join(t.TempDir(), "s.sock"))
	require.NoError(t, err)
	defer l.Close()

	accepted := make(chan net.Conn, 1)
	go func() {
		c, _ := l.Accept()
		accepted <- c
	}()

	client, err := net.Dial("unix", l.Addr().String())
	require.NoError(t, err)
	defer client.Close()

	server := <-accepted
	defer server.Close()

	raw, err := client.(*net.UnixConn).File()
	require.NoError(t, err)
	defer raw.Close()

	fn := SocketShutdown(network{}.Shutdown)
	require.Equal(t, syscall.Errno(0), fn(context.Background(), ffi.Native{}, int32(raw.Fd()), 0x3))

	server.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, err = server.Read(make([]byte, 1))
	require.ErrorIs(t, err, io.EOF)
}
