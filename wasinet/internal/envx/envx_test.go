package envx_test

import (
	"testing"
	"time"

	"github.com/egdaemon/wasinet/wasinet/internal/envx"
	"github.com/stretchr/testify/require"
)

func TestDuration(t *testing.T) {
	t.Run("unset returns the fallback", func(t *testing.T) {
		require.Equal(t, 3*time.Second, envx.Duration(3*time.Second, "WASINET_TEST_DURATION_UNSET"))
	})

	t.Run("valid value is parsed", func(t *testing.T) {
		t.Setenv("WASINET_TEST_DURATION", "1m30s")
		require.Equal(t, 90*time.Second, envx.Duration(time.Second, "WASINET_TEST_DURATION"))
	})

	t.Run("zero overrides a non zero fallback", func(t *testing.T) {
		t.Setenv("WASINET_TEST_DURATION", "0")
		require.Equal(t, time.Duration(0), envx.Duration(time.Second, "WASINET_TEST_DURATION"))
	})

	t.Run("invalid value returns the fallback", func(t *testing.T) {
		t.Setenv("WASINET_TEST_DURATION", "soon")
		require.Equal(t, time.Second, envx.Duration(time.Second, "WASINET_TEST_DURATION"))
	})

	t.Run("first valid key wins", func(t *testing.T) {
		t.Setenv("WASINET_TEST_DURATION_A", "bad")
		t.Setenv("WASINET_TEST_DURATION_B", "2s")
		t.Setenv("WASINET_TEST_DURATION_C", "5s")
		require.Equal(t, 2*time.Second, envx.Duration(time.Second, "WASINET_TEST_DURATION_A", "WASINET_TEST_DURATION_B", "WASINET_TEST_DURATION_C"))
	})
}
