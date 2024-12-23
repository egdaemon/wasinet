package wasinet_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/egdaemon/wasinet/internal/testx"
	"github.com/egdaemon/wasinet/wasinet"

	"github.com/stretchr/testify/assert"
)

func checkTransfer(ctx context.Context, t testing.TB, li addrconn) {
	var (
		buf []byte = make([]byte, 128)
	)

	conn, err := wasinet.DialContext(ctx, li.Addr().Network(), li.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Fatal(err)
		}
	})

	testx.Must(conn.Write([]byte("hello world")))(t)
	n := testx.Must(conn.Read(buf))(t)
	assert.Equal(t, testx.IOString(bytes.NewReader(buf[:n])), "hello world", "expected strings to match")
}

func TestTransferTCPIPv4(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()
	checkTransfer(ctx, t, listentcp(t, "tcp", ":0"))
}

func TestTransferTCP4IPv4(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()
	checkDial(ctx, t, listentcp(t, "tcp4", ":0"))
}

func TestTransferTCPIPv6(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()
	checkDial(ctx, t, listentcp(t, "tcp", "[::]:0"))
}

func TestTransferTCP6IPv6(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()
	checkDial(ctx, t, listentcp(t, "tcp6", "[::]:0"))
}
