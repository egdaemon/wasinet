package wasinet_test

import (
	"testing"
	"time"

	"github.com/egdaemon/wasinet/wasinet"
	"github.com/stretchr/testify/require"
)

func TestInsecureHTTPTimeouts(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		tr := wasinet.InsecureHTTP()
		require.Equal(t, time.Duration(0), tr.ResponseHeaderTimeout)
		require.Equal(t, 10*time.Second, tr.TLSHandshakeTimeout)
		require.Equal(t, 30*time.Second, tr.IdleConnTimeout)
		require.True(t, tr.TLSClientConfig.InsecureSkipVerify)
	})

	t.Run("environment overrides", func(t *testing.T) {
		t.Setenv("WASINET_HIJACK_RESPONSE_HEADER_TIMEOUT", "5m")
		t.Setenv("WASINET_HIJACK_TLS_HANDSHAKE_TIMEOUT", "3s")
		t.Setenv("WASINET_HIJACK_IDLE_CONN_TIMEOUT", "1s")
		tr := wasinet.InsecureHTTP()
		require.Equal(t, 5*time.Minute, tr.ResponseHeaderTimeout)
		require.Equal(t, 3*time.Second, tr.TLSHandshakeTimeout)
		require.Equal(t, time.Second, tr.IdleConnTimeout)
	})
}
