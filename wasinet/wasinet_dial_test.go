package wasinet_test

import (
	"context"
	"testing"

	"github.com/egdaemon/wasinetruntime/internal/testx"
	"github.com/egdaemon/wasinetruntime/wasinet"
)

func checkDial(ctx context.Context, t testing.TB, li addrconn) {
	conn, err := wasinet.DialContext(ctx, li.Addr().Network(), li.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Fatal(err)
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
