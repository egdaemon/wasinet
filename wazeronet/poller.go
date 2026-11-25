//go:build !wasip1

package wazeronet

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/experimental/sys"
	"golang.org/x/sys/unix"
)

// WithPoller has poll_oneoff wait on the host sockets the guest opened through wasinet, which the
// module's file table does not hold, so a goroutine waiting on one parks in the guest's netpoller
// until the socket is ready.
func WithPoller(ctx context.Context) context.Context {
	return experimental.WithPoller(ctx, poll)
}

func poll(fds []experimental.PollFd, timeout time.Duration) sys.Errno {
	var (
		native = make([]unix.PollFd, len(fds))
		millis = -1
		err    error
	)

	for i, fd := range fds {
		native[i] = unix.PollFd{Fd: fd.Fd}
		if fd.Events&sys.POLLIN != 0 {
			native[i].Events |= unix.POLLIN
		}
		if fd.Events&sys.POLLOUT != 0 {
			native[i].Events |= unix.POLLOUT
		}
	}

	if timeout >= 0 {
		millis = int(min((timeout+time.Millisecond-1)/time.Millisecond, math.MaxInt32))
	}

	for _, err = unix.Poll(native, millis); errors.Is(err, unix.EINTR); _, err = unix.Poll(native, millis) {
	}
	if err != nil {
		return sys.UnwrapOSError(err)
	}

	// an error or hang up on the socket wakes the guest too, whose next call on it reports it.
	for i := range native {
		if native[i].Revents != 0 {
			fds[i].Revents = fds[i].Events
		}
	}

	return 0
}
