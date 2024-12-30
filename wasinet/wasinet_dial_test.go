package wasinet_test

import (
	"context"
	"log"
	"net"
	"syscall"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/egdaemon/wasinet/wasinet"
	"github.com/egdaemon/wasinet/wasinet/testx"
	"github.com/stretchr/testify/require"
)

func checkDial(ctx context.Context, t testing.TB, li addrconn) {
	conn, err := wasinet.DialContext(ctx, li.Addr().Network(), li.Addr().String())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})
}

func checkDialErr(ctx context.Context, t testing.TB, li addrconn, expected *net.OpError) {
	actual := new(net.OpError)
	log.Println("sigh", li, li.Addr().Network(), li.Addr().String())
	conn, err := wasinet.DialContext(ctx, li.Addr().Network(), li.Addr().String())
	log.Printf("error %T - %s\n", err, spew.Sdump(err))
	require.ErrorAs(t, err, &actual)

	require.Equal(t, expected.Net, actual.Net)
	require.Equal(t, expected.Op, actual.Op)
	// require.Equal(t, expected.Source, actual.Source)
	// require.Equal(t, expected.Addr, actual.Addr)
	require.Equal(t, expected.Err, actual.Err)
	t.Cleanup(func() {
		if conn != nil {
			require.NoError(t, conn.Close())
		}
	})
}

func TestDialTCPIPv4(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()
	checkDial(ctx, t, listentcp(t, "tcp", ":0"))
}

func TestDialTCP4IPv4(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()
	checkDial(ctx, t, listentcp(t, "tcp4", ":0"))
}

func TestDialTCPIPv6(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()
	checkDial(ctx, t, listentcp(t, "tcp", "[::]:0"))
}

func TestDialTCP6IPv6(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()
	checkDial(ctx, t, listentcp(t, "tcp6", "[::]:0"))
}

func TestDialUDPIPv4(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()
	checkDial(ctx, t, listenudp(t, "udp", ":0"))
}

func TestDialUDP4IPv4(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()
	checkDial(ctx, t, listenudp(t, "udp4", ":0"))
}

func TestDialUDPIPv6(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()
	checkDial(ctx, t, listenudp(t, "udp", "[::]:0"))
}

func TestDialTCPNoService(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	require.NoError(t, l.Close())
	// close tcp 0.0.0.0:45767: use of closed network connection
	checkDialErr(ctx, t, l, &net.OpError{Op: "dial", Net: "tcp", Addr: l.Addr(), Err: syscall.ECONNREFUSED})
}
