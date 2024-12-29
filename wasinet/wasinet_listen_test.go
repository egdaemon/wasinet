package wasinet_test

import (
	"testing"

	"github.com/egdaemon/wasinet/wasinet"
	"github.com/egdaemon/wasinet/wasinet/testx"
	"github.com/stretchr/testify/require"
)

func checkListen(t *testing.T, network, address string) {
	ctx, done := testx.WithDeadline(t)
	defer done()
	li, err := wasinet.Listen(ctx, network, address)
	if err != nil {
		t.Fatal(err)
	}

	if err := li.Close(); err != nil {
		t.Fatal(err)
	}
}

func checkListenPacket(t *testing.T, network, address string) {
	ctx, done := testx.WithDeadline(t)
	defer done()
	li, err := wasinet.ListenPacket(ctx, network, address)
	require.NoError(t, err)
	require.NoError(t, li.Close())
}

func TestListenTCPIPv4(t *testing.T) {
	checkListen(t, "tcp", ":0")
}

func TestListenTCP4IPv4(t *testing.T) {
	checkListen(t, "tcp4", ":0")
}

func TestListenTCPIPv6(t *testing.T) {
	checkListen(t, "tcp", "[::]:0")
}

func TestListenTCP6IPv6(t *testing.T) {
	checkListen(t, "tcp6", "[::]:0")
}

func TestListenUDPIPv4(t *testing.T) {
	checkListenPacket(t, "udp", ":0")
}

func TestListenUDP4IPv4(t *testing.T) {
	checkListenPacket(t, "udp4", ":0")
}

func TestListenUDPIPv6(t *testing.T) {
	checkListenPacket(t, "udp", "[::]:0")
}

func TestListenUDP6IPv6(t *testing.T) {
	checkListenPacket(t, "udp6", "[::]:0")
}
