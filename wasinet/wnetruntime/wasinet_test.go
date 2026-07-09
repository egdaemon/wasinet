//go:build !wasip1 && !windows

package wnetruntime

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func inet4(ip string) *unix.SockaddrInet4 {
	return &unix.SockaddrInet4{Addr: netip.MustParseAddr(ip).As4()}
}

func inet6(ip string) *unix.SockaddrInet6 {
	return &unix.SockaddrInet6{Addr: netip.MustParseAddr(ip).As16()}
}

func TestSockaddrAddr(t *testing.T) {
	t.Run("inet4", func(t *testing.T) {
		addr, ok := sockaddrAddr(inet4("192.168.1.1"))
		require.True(t, ok)
		require.Equal(t, netip.MustParseAddr("192.168.1.1"), addr)
	})

	t.Run("inet6", func(t *testing.T) {
		addr, ok := sockaddrAddr(inet6("::1"))
		require.True(t, ok)
		require.Equal(t, netip.MustParseAddr("::1"), addr)
	})

	t.Run("unix", func(t *testing.T) {
		_, ok := sockaddrAddr(&unix.SockaddrUnix{Name: "/tmp/foo"})
		require.False(t, ok)
	})

	t.Run("nil", func(t *testing.T) {
		_, ok := sockaddrAddr(nil)
		require.False(t, ok)
	})
}

func TestNetworkRestricted(t *testing.T) {
	t.Run("no lists permits everything", func(t *testing.T) {
		n := network{}
		require.NoError(t, n.restricted(inet4("8.8.8.8")))
	})

	t.Run("block list rejects matching address", func(t *testing.T) {
		n := network{block: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}}
		require.ErrorIs(t, n.restricted(inet4("10.1.2.3")), unix.EACCES)
	})

	t.Run("block list permits non matching address", func(t *testing.T) {
		n := network{block: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}}
		require.NoError(t, n.restricted(inet4("8.8.8.8")))
	})

	t.Run("allow list takes precedence over block list", func(t *testing.T) {
		n := network{
			allow: []netip.Prefix{netip.MustParsePrefix("10.1.0.0/16")},
			block: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")},
		}
		require.NoError(t, n.restricted(inet4("10.1.2.3")))
	})

	t.Run("outside allow still checked against block", func(t *testing.T) {
		n := network{
			allow: []netip.Prefix{netip.MustParsePrefix("10.2.0.0/16")},
			block: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")},
		}
		require.ErrorIs(t, n.restricted(inet4("10.1.2.3")), unix.EACCES)
	})

	t.Run("non inet sockaddr is never restricted", func(t *testing.T) {
		n := network{block: []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0")}}
		require.NoError(t, n.restricted(&unix.SockaddrUnix{Name: "/tmp/foo"}))
	})

	t.Run("ipv6", func(t *testing.T) {
		n := network{block: []netip.Prefix{netip.MustParsePrefix("fc00::/7")}}
		require.ErrorIs(t, n.restricted(inet6("fd00::1")), unix.EACCES)
		require.NoError(t, n.restricted(inet6("2001:4860:4860::8888")))
	})
}

func TestPublicOnlyBlocksPrivateRanges(t *testing.T) {
	n := network{block: privatePrefixes}

	blocked4 := []string{
		"10.1.2.3", "172.16.0.1", "192.168.1.1", "127.0.0.1",
		"169.254.1.1", "224.0.0.1", "100.64.0.1", "0.0.0.0",
	}
	for _, ip := range blocked4 {
		require.ErrorIsf(t, n.restricted(inet4(ip)), unix.EACCES, "expected %s to be blocked", ip)
	}

	permitted4 := []string{"8.8.8.8", "1.1.1.1", "93.184.216.34"}
	for _, ip := range permitted4 {
		require.NoErrorf(t, n.restricted(inet4(ip)), "expected %s to be permitted", ip)
	}

	blocked6 := []string{"::1", "fe80::1", "fd00::1", "ff02::1"}
	for _, ip := range blocked6 {
		require.ErrorIsf(t, n.restricted(inet6(ip)), unix.EACCES, "expected %s to be blocked", ip)
	}

	require.NoError(t, n.restricted(inet6("2001:4860:4860::8888")))
}
