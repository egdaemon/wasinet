//go:build !wasip1 && !windows

package wnetruntime

import (
	"context"
	"fmt"
	"net"
	"syscall"
	"testing"

	"github.com/egdaemon/wasinet/wasinet/ffi"
	"github.com/egdaemon/wasinet/wasinet/stdlib/wasip1syscall"
	"github.com/stretchr/testify/require"
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
