package wasip1syscall

import (
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrnoTranslate(t *testing.T) {
	// the wasi preview 1 numbering, which differs from every host's.
	t.Run("a peer that went away maps to wasi's codes", func(t *testing.T) {
		require.Equal(t, syscall.Errno(64), ErrnoTranslate(syscall.EPIPE))
		require.Equal(t, syscall.Errno(15), ErrnoTranslate(syscall.ECONNRESET))
	})

	t.Run("success stays success", func(t *testing.T) {
		require.Equal(t, syscall.Errno(0), ErrnoTranslate(syscall.Errno(0)))
	})
}
