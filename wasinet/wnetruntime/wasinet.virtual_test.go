//go:build !wasip1 && !windows

package wnetruntime

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

type dialCall struct{ network, address string }

type recordingDialer struct {
	mu    sync.Mutex
	calls []dialCall
	conn  net.Conn
	err   error
}

func (d *recordingDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	d.mu.Lock()
	d.calls = append(d.calls, dialCall{network, address})
	d.mu.Unlock()
	return d.conn, d.err
}

type recordingPacketDialer struct {
	mu    sync.Mutex
	calls []dialCall
}

func (d *recordingPacketDialer) ListenPacket(ctx context.Context, network, address string) (net.PacketConn, error) {
	d.mu.Lock()
	d.calls = append(d.calls, dialCall{network, address})
	d.mu.Unlock()
	return net.ListenPacket("udp", "127.0.0.1:0")
}

type fakeResolver struct {
	hosts []string
	err   error
}

func (r fakeResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return r.hosts, r.err
}

func sendToWithRetry(t *testing.T, ctx context.Context, v Socket, fd int, sa unix.Sockaddr, data []byte) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, err := v.SendTo(ctx, fd, sa, [][]byte{data}, nil, 0)
		if err == nil {
			return
		}
		if err == unix.EAGAIN {
			time.Sleep(2 * time.Millisecond)
			continue
		}
		t.Fatalf("SendTo failed: %v", err)
	}
	t.Fatal("SendTo timed out")
}

func recvFromWithRetry(t *testing.T, ctx context.Context, v Socket, fd int, buf []byte) int {
	t.Helper()
	vecs := [][]byte{buf}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		n, _, _, err := v.RecvFrom(ctx, fd, vecs, nil, 0)
		if err == nil {
			return n
		}
		if err == unix.EAGAIN {
			time.Sleep(2 * time.Millisecond)
			continue
		}
		t.Fatalf("RecvFrom failed: %v", err)
	}
	t.Fatal("RecvFrom timed out")
	return 0
}

func TestVirtualConnectDialsPermittedDestination(t *testing.T) {
	client, _ := net.Pipe()
	dialer := &recordingDialer{conn: client}
	packets := &recordingPacketDialer{}
	fw := Firewall{Allow: []netip.Prefix{netip.MustParsePrefix("93.184.216.0/24")}}
	v := Virtual(dialer, packets, net.DefaultResolver, fw)
	ctx := context.Background()

	fd, err := v.Open(ctx, WASI_AF_INET, unix.SOCK_STREAM, 0)
	require.NoError(t, err)

	sa := inet4("93.184.216.34")
	sa.Port = 80
	require.NoError(t, v.Connect(ctx, fd, sa))

	require.Len(t, dialer.calls, 1)
	require.Equal(t, "tcp", dialer.calls[0].network)
	require.Equal(t, "93.184.216.34:80", dialer.calls[0].address)
}

func TestVirtualConnectNeverDialsBlockedDestination(t *testing.T) {
	dialer := &recordingDialer{}
	packets := &recordingPacketDialer{}
	fw := Firewall{Block: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}}
	v := Virtual(dialer, packets, net.DefaultResolver, fw)
	ctx := context.Background()

	fd, err := v.Open(ctx, WASI_AF_INET, unix.SOCK_STREAM, 0)
	require.NoError(t, err)

	sa := inet4("10.1.2.3")
	sa.Port = 80
	err = v.Connect(ctx, fd, sa)
	require.ErrorIs(t, err, unix.EACCES)
	require.Len(t, dialer.calls, 0)
}

func TestVirtualStreamFdLifecycle(t *testing.T) {
	client, server := net.Pipe()
	dialer := &recordingDialer{conn: client}
	packets := &recordingPacketDialer{}
	fw := Firewall{Allow: []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0")}}
	v := Virtual(dialer, packets, net.DefaultResolver, fw)
	ctx := context.Background()

	fd, err := v.Open(ctx, WASI_AF_INET, unix.SOCK_STREAM, 0)
	require.NoError(t, err)

	sa := inet4("1.2.3.4")
	sa.Port = 9
	require.NoError(t, v.Connect(ctx, fd, sa))
	require.Len(t, dialer.calls, 1)

	readDone := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 4)
		n, _ := server.Read(buf)
		readDone <- buf[:n]
	}()
	sendToWithRetry(t, ctx, v, fd, nil, []byte("ping"))
	require.Equal(t, "ping", string(<-readDone))

	writeDone := make(chan struct{})
	go func() {
		server.Write([]byte("pong"))
		close(writeDone)
	}()
	buf := make([]byte, 4)
	n := recvFromWithRetry(t, ctx, v, fd, buf)
	require.Equal(t, 4, n)
	require.Equal(t, "pong", string(buf[:n]))
	<-writeDone

	require.NoError(t, v.Shutdown(ctx, fd, unix.SHUT_RDWR))

	_, _, _, err = v.RecvFrom(ctx, fd, [][]byte{buf}, nil, 0)
	require.ErrorIs(t, err, unix.EBADF)
}

func TestVirtualPacketDialerDeferred(t *testing.T) {
	dialer := &recordingDialer{}
	packets := &recordingPacketDialer{}
	fw := Firewall{Allow: []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0")}}
	v := Virtual(dialer, packets, net.DefaultResolver, fw)
	ctx := context.Background()

	fd, err := v.Open(ctx, WASI_AF_INET, unix.SOCK_DGRAM, 0)
	require.NoError(t, err)
	require.Len(t, packets.calls, 0)

	_, err = v.LocalAddr(ctx, fd)
	require.NoError(t, err)
	require.Len(t, packets.calls, 1)

	_, err = v.LocalAddr(ctx, fd)
	require.NoError(t, err)
	require.Len(t, packets.calls, 1)
}

func TestVirtualDatagramSendToEnforcesFirewallPerCall(t *testing.T) {
	peer, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	defer peer.Close()
	peerAddr := peer.LocalAddr().(*net.UDPAddr)

	dialer := &recordingDialer{}
	packets := &recordingPacketDialer{}
	fw := Firewall{
		Allow: []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")},
		Block: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")},
	}
	v := Virtual(dialer, packets, net.DefaultResolver, fw)
	ctx := context.Background()

	fd, err := v.Open(ctx, WASI_AF_INET, unix.SOCK_DGRAM, 0)
	require.NoError(t, err)

	permitted := inet4("127.0.0.1")
	permitted.Port = peerAddr.Port
	readDone := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 4)
		n, _, _ := peer.ReadFrom(buf)
		readDone <- buf[:n]
	}()
	sendToWithRetry(t, ctx, v, fd, permitted, []byte("ping"))
	require.Equal(t, "ping", string(<-readDone))

	blocked := inet4("10.0.0.5")
	blocked.Port = 9999
	_, err = v.SendTo(ctx, fd, blocked, [][]byte{[]byte("nope")}, nil, 0)
	require.ErrorIs(t, err, unix.EACCES)
}

func TestVirtualDatagramConnectSetsDefaultPeer(t *testing.T) {
	peer, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	defer peer.Close()
	peerAddr := peer.LocalAddr().(*net.UDPAddr)

	dialer := &recordingDialer{}
	packets := &recordingPacketDialer{}
	fw := Firewall{
		Allow: []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")},
		Block: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")},
	}
	v := Virtual(dialer, packets, net.DefaultResolver, fw)
	ctx := context.Background()

	fd, err := v.Open(ctx, WASI_AF_INET, unix.SOCK_DGRAM, 0)
	require.NoError(t, err)

	sa := inet4("127.0.0.1")
	sa.Port = peerAddr.Port
	require.NoError(t, v.Connect(ctx, fd, sa))

	readDone := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 4)
		n, _, _ := peer.ReadFrom(buf)
		readDone <- buf[:n]
	}()
	sendToWithRetry(t, ctx, v, fd, nil, []byte("ping"))
	require.Equal(t, "ping", string(<-readDone))

	fd2, err := v.Open(ctx, WASI_AF_INET, unix.SOCK_DGRAM, 0)
	require.NoError(t, err)

	blocked := inet4("10.0.0.5")
	blocked.Port = 9999
	err = v.Connect(ctx, fd2, blocked)
	require.ErrorIs(t, err, unix.EACCES)

	_, err = v.SendTo(ctx, fd2, nil, [][]byte{[]byte("x")}, nil, 0)
	require.ErrorIs(t, err, unix.ENOTCONN)
}

func TestVirtualAddrIPUsesInjectedResolver(t *testing.T) {
	dialer := &recordingDialer{}
	packets := &recordingPacketDialer{}
	ctx := context.Background()

	v := Virtual(dialer, packets, fakeResolver{hosts: []string{"93.184.216.34", "2606:2800:220:1:248:1893:25c8:1946"}}, Firewall{})
	ips, err := v.AddrIP(ctx, "ip4", "example.com")
	require.NoError(t, err)
	require.Len(t, ips, 1)
	require.Equal(t, "93.184.216.34", ips[0].String())

	errBoom := errors.New("boom")
	v2 := Virtual(dialer, packets, fakeResolver{err: errBoom}, Firewall{})
	_, err = v2.AddrIP(ctx, "ip4", "example.com")
	require.ErrorIs(t, err, errBoom)
}

func TestVirtualUnsupportedOps(t *testing.T) {
	dialer := &recordingDialer{}
	packets := &recordingPacketDialer{}
	v := Virtual(dialer, packets, net.DefaultResolver, Firewall{})
	ctx := context.Background()

	fd, err := v.Open(ctx, WASI_AF_INET, unix.SOCK_STREAM, 0)
	require.NoError(t, err)

	require.ErrorIs(t, v.Bind(ctx, fd, inet4("1.2.3.4")), syscall.ENOTSUP)
	require.ErrorIs(t, v.Listen(ctx, fd, 5), syscall.ENOTSUP)
	_, _, err = v.Accept(ctx, fd)
	require.ErrorIs(t, err, syscall.ENOTSUP)
}

func TestVirtualRejectsUnixSockaddr(t *testing.T) {
	dialer := &recordingDialer{}
	packets := &recordingPacketDialer{}
	fw := Firewall{Allow: []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0")}}
	v := Virtual(dialer, packets, net.DefaultResolver, fw)
	ctx := context.Background()

	fd, err := v.Open(ctx, WASI_AF_INET, unix.SOCK_STREAM, 0)
	require.NoError(t, err)

	err = v.Connect(ctx, fd, &unix.SockaddrUnix{Name: "/tmp/x"})
	require.ErrorIs(t, err, unix.EAFNOSUPPORT)
	require.Len(t, dialer.calls, 0)
}
