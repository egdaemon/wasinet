package wasinet_test

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/egdaemon/wasinet/wasinet"
	"github.com/egdaemon/wasinet/wasinet/internal/bytesx"
	"github.com/egdaemon/wasinet/wasinet/testx"

	"github.com/stretchr/testify/require"
)

func checkTransfer(ctx context.Context, t testing.TB, li addrconn, amount int64) {
	var (
		serr       error
		amountsent int64
	)

	conn, err := wasinet.DialContext(ctx, li.Addr().Network(), li.Addr().String())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})

	digestsent := md5.New()
	digestrecv := md5.New()

	go func() {
		amountsent, serr = io.CopyN(conn, io.TeeReader(rand.Reader, digestsent), amount)
	}()

	n := testx.Must(io.Copy(digestrecv, io.LimitReader(conn, amount)))(t)
	require.Equal(t, amount, n)
	require.Equal(t, amount, amountsent)
	require.NoError(t, err)
	require.NoError(t, serr)
	require.Equal(t, digestsent.Sum(nil), digestrecv.Sum(nil), "expected digests to match")
}

func TestTransferTCPIPv4(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()
	checkTransfer(ctx, t, listenstream(t, "tcp", ":0"), bytesx.KiB)
}

func TestTransferTCP4IPv4(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()
	checkTransfer(ctx, t, listenstream(t, "tcp4", ":0"), bytesx.KiB)
}

func TestTransferTCPIPv6(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()
	checkTransfer(ctx, t, listenstream(t, "tcp", "[::]:0"), bytesx.KiB)
}

func TestTransferTCP6IPv6(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()
	checkTransfer(ctx, t, listenstream(t, "tcp6", "[::]:0"), bytesx.KiB)
}

func TestTransferTCPLarge16MB(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()
	checkTransfer(ctx, t, listenstream(t, "tcp", ":0"), 16*bytesx.MiB)
}

func TestTransferUnix(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()
	checkTransfer(ctx, t, listenstream(t, "unix", filepath.Join(t.TempDir(), "test.socket")), bytesx.KiB)
}

func TestTransferUnixLarge(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()
	checkTransfer(ctx, t, listenstream(t, "unix", filepath.Join(t.TempDir(), "test.socket")), 16*bytesx.MiB)
}

func TestTransferHTTP(t *testing.T) {
	var buf bytes.Buffer

	n, err := io.CopyN(&buf, rand.Reader, 16*bytesx.KiB)
	require.NoError(t, err)
	require.Equal(t, n, int64(16*bytesx.KiB))

	m := http.NewServeMux()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.Copy(w, bytes.NewBuffer(buf.Bytes())); err != nil {
			log.Println("copy failed", err)
			return
		}
	})
	svc := httptest.NewServer(m)

	c := httpclient()
	rsp, err := c.Get(svc.URL)
	require.NoError(t, err, "expected request to succeed")
	require.Equal(t, rsp.StatusCode, http.StatusOK)
	received, err := io.ReadAll(rsp.Body)
	require.NoError(t, err)
	require.Equal(t, buf.Bytes(), received)
}

func TestTransferHTTPExternal(t *testing.T) {
	c := httpclient()

	rsp, err := c.Get("https://google.com")
	require.NoError(t, err, "expected request to succeed")
	require.Equal(t, rsp.StatusCode, http.StatusOK)
	bdy, err := io.ReadAll(rsp.Body)
	require.NoError(t, err, "expected request to read body")
	// log.Println(string(bdy))
	require.Greater(t, len(bdy), 10)
}
