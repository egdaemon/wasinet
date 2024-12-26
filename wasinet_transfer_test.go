package wasinet_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/egdaemon/wasinet"
	"github.com/egdaemon/wasinet/internal/bytesx"
	"github.com/egdaemon/wasinet/internal/testx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	rsp, err := c.Get("http://google.com")
	require.NoError(t, err, "expected request to succeed")
	require.Equal(t, rsp.StatusCode, http.StatusOK)
	bdy, err := io.ReadAll(rsp.Body)
	require.NoError(t, err, "expected request to read body")
	log.Println("DRP", string(bdy))
}
