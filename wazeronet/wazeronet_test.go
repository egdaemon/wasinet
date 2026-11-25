package wazeronet_test

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"io"
	"log"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/egdaemon/wasinet/wasinet/testx"
	"github.com/egdaemon/wasinet/wasinet/wnetruntime"
	"github.com/egdaemon/wasinet/wazeronet"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func compile(ctx context.Context, in string, output string) (err error) {
	cmd := exec.CommandContext(ctx, "go", "build", "-trimpath", "-o", output, in)
	cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func compileAndRun(ctx context.Context, t *testing.T, path string, n wnetruntime.Socket, cfg func(wazero.ModuleConfig) wazero.ModuleConfig) error {
	// ctx = experimental.WithFunctionListenerFactory(ctx,
	// 	logging.NewHostLoggingListenerFactory(os.Stderr, logging.LogScopeAll))
	ctx = wazeronet.WithPoller(ctx)

	// Create a new WebAssembly Runtime.
	runtime := wazero.NewRuntimeWithConfig(
		ctx,
		wazero.NewRuntimeConfig().WithDebugInfoEnabled(true),
	)

	mcfg := wazero.NewModuleConfig().WithStdin(
		os.Stdin,
	).WithStderr(
		os.Stderr,
	).WithStdout(
		os.Stdout,
	).WithSysNanotime().WithSysWalltime().WithRandSource(rand.Reader)

	mcfg = cfg(mcfg)

	wasienv, err := wasi_snapshot_preview1.NewBuilder(runtime).Instantiate(ctx)
	if err != nil {
		return err
	}
	defer wasienv.Close(ctx)

	wasinet, err := wazeronet.Module(runtime, n).Instantiate(ctx)
	if err != nil {
		return err
	}
	defer wasinet.Close(ctx)

	compiled := filepath.Join(t.TempDir(), "main.wasm")
	if err = compile(ctx, path, compiled); err != nil {
		return err
	}

	wasi, err := os.ReadFile(compiled)
	if err != nil {
		return err
	}

	c, err := runtime.CompileModule(ctx, wasi)
	if err != nil {
		return err
	}
	defer c.Close(ctx)

	m, err := runtime.InstantiateModule(ctx, c, mcfg.WithName(path))
	if err != nil {
		return err
	}
	defer m.Close(ctx)

	return nil
}

func TestNetworkExample1(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()

	require.NoError(t, compileAndRun(ctx, t, testx.Fixture("example1", "main.go"), wnetruntime.Unrestricted(), func(mc wazero.ModuleConfig) wazero.ModuleConfig { return mc }))
}

func TestUnix(t *testing.T) {
	transfer := func(t *testing.T, amount int) {
		ctx, done := testx.WithDeadline(t)
		defer done()

		ctx, done = context.WithTimeout(ctx, 30*time.Second)
		defer done()

		tmpdir := t.TempDir()
		li, err := net.Listen("unix", filepath.Join(tmpdir, "socket"))
		require.NoError(t, err)
		defer li.Close()

		go func() {
			var (
				err  error
				conn net.Conn
			)
			for conn, err = li.Accept(); err == nil; conn, err = li.Accept() {
				server, client := net.Pipe()
				go func(c net.Conn) {
					if _, err := io.Copy(c, server); err != nil {
						log.Println("server copy failed", err)
					}
				}(conn)
				go func(c net.Conn) {
					defer c.Close()
					if _, err := io.Copy(client, c); err != nil {
						log.Println("client copy failed", err)
					}
				}(conn)
			}
		}()

		n := wnetruntime.Unrestricted(wnetruntime.OptionFSPrefixes(wnetruntime.FSPrefix{Host: tmpdir, Guest: "/test"}))

		require.NoError(t, compileAndRun(ctx, t, testx.Fixture("example2", "main.go"), n, func(mc wazero.ModuleConfig) wazero.ModuleConfig {
			return mc.WithFSConfig(
				wazero.NewFSConfig().WithDirMount(
					tmpdir, "/test",
				),
			).WithEnv("TRANSFER_AMOUNT", strconv.Itoa(amount))
		}))
	}

	t.Run("example1", func(t *testing.T) {
		transfer(t, 1024)
	})

	// writes beyond the socket's send buffer (8 KiB by default on macOS) are partial or EAGAIN.
	t.Run("16MiB", func(t *testing.T) {
		transfer(t, 16<<20)
	})

	t.Run("guest listens", func(t *testing.T) {
		var (
			conn   net.Conn
			err    error
			echoed = make([]byte, 5)
		)
		ctx, done := testx.WithDeadline(t)
		defer done()

		ctx, done = context.WithTimeout(ctx, 30*time.Second)
		defer done()

		tmpdir := t.TempDir()
		socket := filepath.Join(tmpdir, "listen.socket")
		n := wnetruntime.Unrestricted(wnetruntime.OptionFSPrefixes(wnetruntime.FSPrefix{Host: tmpdir, Guest: "/test"}))

		guest := make(chan error, 1)
		go func() {
			guest <- compileAndRun(ctx, t, testx.Fixture("example4", "main.go"), n, func(mc wazero.ModuleConfig) wazero.ModuleConfig {
				return mc.WithFSConfig(wazero.NewFSConfig().WithDirMount(tmpdir, "/test"))
			})
		}()

		// the guest compiles before it listens, so dial until the socket appears.
		for conn, err = net.Dial("unix", socket); err != nil; conn, err = net.Dial("unix", socket) {
			select {
			case cause := <-guest:
				require.NoError(t, cause)
				require.FailNow(t, "guest exited without listening")
			case <-time.After(10 * time.Millisecond):
			}
		}
		defer conn.Close()

		_, err = conn.Write([]byte("hello"))
		require.NoError(t, err)
		_, err = io.ReadFull(conn, echoed)
		require.NoError(t, err)
		require.Equal(t, "hello", string(echoed))
		require.NoError(t, <-guest)
	})
}

func TestUnixHalfClose(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()

	ctx, done = context.WithTimeout(ctx, 30*time.Second)
	defer done()

	tmpdir := t.TempDir()
	li, err := net.Listen("unix", filepath.Join(tmpdir, "socket"))
	require.NoError(t, err)
	defer li.Close()

	// echoes until the guest half closes, so the guest only sees EOF if its CloseWrite reached the host.
	go func() {
		conn, err := li.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		if _, err = io.Copy(conn, conn); err != nil {
			log.Println("echo failed", err)
		}
	}()

	n := wnetruntime.Unrestricted(wnetruntime.OptionFSPrefixes(wnetruntime.FSPrefix{Host: tmpdir, Guest: "/test"}))
	require.NoError(t, compileAndRun(ctx, t, testx.Fixture("example5", "main.go"), n, func(mc wazero.ModuleConfig) wazero.ModuleConfig {
		return mc.WithFSConfig(wazero.NewFSConfig().WithDirMount(tmpdir, "/test"))
	}))
}

// listenloopback starts a host-side (non-wasi) tcp listener that pipes any
// accepted connection back to itself, mirroring TestUnix's setup. it exists
// purely so the wasm guest has something local to dial.
func listenloopback(t *testing.T) net.Listener {
	li, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { li.Close() })

	go func() {
		var (
			err  error
			conn net.Conn
		)
		for conn, err = li.Accept(); err == nil; conn, err = li.Accept() {
			server, client := net.Pipe()
			go func(c net.Conn) {
				if _, err := io.Copy(c, server); err != nil {
					log.Println("server copy failed", err)
				}
			}(conn)
			go func(c net.Conn) {
				defer c.Close()
				if _, err := io.Copy(client, c); err != nil {
					log.Println("client copy failed", err)
				}
			}(conn)
		}
	}()

	return li
}

func withDialAddr(addr string) func(wazero.ModuleConfig) wazero.ModuleConfig {
	return func(mc wazero.ModuleConfig) wazero.ModuleConfig {
		return mc.WithEnv("DIAL_ADDR", addr)
	}
}

func TestNetworkBlockList(t *testing.T) {
	t.Run("blocked prefix rejects the connection", func(t *testing.T) {
		ctx, done := testx.WithDeadline(t)
		defer done()

		li := listenloopback(t)
		n := wnetruntime.New(wnetruntime.OptionFirewall(wnetruntime.NewFirewall(wnetruntime.FirewallOptionBlock(netip.MustParsePrefix("127.0.0.0/8")))))

		require.Error(t, compileAndRun(ctx, t, testx.Fixture("example3", "main.go"), n, withDialAddr(li.Addr().String())))
	})

	t.Run("unrelated block prefix permits the connection", func(t *testing.T) {
		ctx, done := testx.WithDeadline(t)
		defer done()

		li := listenloopback(t)
		n := wnetruntime.New(wnetruntime.OptionFirewall(wnetruntime.NewFirewall(wnetruntime.FirewallOptionBlock(netip.MustParsePrefix("10.0.0.0/8")))))

		require.NoError(t, compileAndRun(ctx, t, testx.Fixture("example3", "main.go"), n, withDialAddr(li.Addr().String())))
	})

	t.Run("allowed prefix permits the connection", func(t *testing.T) {
		ctx, done := testx.WithDeadline(t)
		defer done()

		li := listenloopback(t)
		n := wnetruntime.New(wnetruntime.OptionFirewall(wnetruntime.NewFirewall(wnetruntime.FirewallOptionAllow(netip.MustParsePrefix("127.0.0.0/8")))))

		require.NoError(t, compileAndRun(ctx, t, testx.Fixture("example3", "main.go"), n, withDialAddr(li.Addr().String())))
	})

	t.Run("allow takes precedence over an overlapping block", func(t *testing.T) {
		ctx, done := testx.WithDeadline(t)
		defer done()

		li := listenloopback(t)
		n := wnetruntime.New(
			wnetruntime.OptionFirewall(wnetruntime.NewFirewall(
				wnetruntime.FirewallOptionAllow(netip.MustParsePrefix("127.0.0.1/32")),
				wnetruntime.FirewallOptionBlock(netip.MustParsePrefix("127.0.0.0/8")),
			)),
		)

		require.NoError(t, compileAndRun(ctx, t, testx.Fixture("example3", "main.go"), n, withDialAddr(li.Addr().String())))
	})

	t.Run("PublicOnly blocks loopback by default", func(t *testing.T) {
		ctx, done := testx.WithDeadline(t)
		defer done()

		li := listenloopback(t)
		n := wnetruntime.PublicOnly()

		require.Error(t, compileAndRun(ctx, t, testx.Fixture("example3", "main.go"), n, withDialAddr(li.Addr().String())))
	})
}

// sockopt is a SetSocketOption call as it arrived at the host, after the wasi values the guest
// sent have been translated to the host's system values.
type sockopt struct {
	level int
	name  int
	value []byte
}

// recordingsocket wraps a Socket and records every SetSocketOption call that crosses the wasi
// boundary before delegating to the wrapped implementation.
type recordingsocket struct {
	wnetruntime.Socket
	mu   sync.Mutex
	opts []sockopt
}

func (t *recordingsocket) SetSocketOption(ctx context.Context, fd int, level, name int, value []byte) error {
	t.mu.Lock()
	t.opts = append(t.opts, sockopt{level: level, name: name, value: append([]byte(nil), value...)})
	t.mu.Unlock()
	return t.Socket.SetSocketOption(ctx, fd, level, name, value)
}

func TestSetsockoptTimeval(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()

	li := listenloopback(t)
	n := &recordingsocket{Socket: wnetruntime.Unrestricted()}

	require.NoError(t, compileAndRun(ctx, t, testx.Fixture("example6", "main.go"), n, withDialAddr(li.Addr().String())))

	type timeval struct {
		name int
		sec  int64
		usec int64
	}

	var got []timeval
	n.mu.Lock()
	for _, o := range n.opts {
		if o.name != syscall.SO_RCVTIMEO && o.name != syscall.SO_SNDTIMEO {
			continue
		}

		require.Equal(t, syscall.SOL_SOCKET, o.level, "timeouts must be set at SOL_SOCKET")
		require.Len(t, o.value, 16, "timeval is two int64s")
		got = append(got, timeval{
			name: o.name,
			sec:  int64(binary.LittleEndian.Uint64(o.value[0:8])),
			usec: int64(binary.LittleEndian.Uint64(o.value[8:16])),
		})
	}
	n.mu.Unlock()

	// the order is the one fixtures/example6 performs the steps in.
	require.Len(t, got, 7)

	// a deadline 1.5s away: whole seconds and the remaining microseconds. the remainder is under
	// the full 500ms by however long the call took, and well above the 999 a millisecond count
	// could ever produce.
	for _, i := range []int{0, 1, 6} {
		require.Equal(t, int64(1), got[i].sec)
		require.Greater(t, got[i].usec, int64(400_000))
		require.LessOrEqual(t, got[i].usec, int64(500_000))
	}
	require.Equal(t, syscall.SO_RCVTIMEO, got[0].name)
	require.Equal(t, syscall.SO_SNDTIMEO, got[1].name)
	require.Equal(t, syscall.SO_RCVTIMEO, got[6].name) // the datagram socket

	// a zero time clears the timeout, which a zero timeval means.
	require.Equal(t, timeval{name: syscall.SO_RCVTIMEO}, got[2])
	require.Equal(t, timeval{name: syscall.SO_SNDTIMEO}, got[3])

	// a deadline in the past must not become zero (no timeout): it is the smallest timeout.
	require.Equal(t, timeval{name: syscall.SO_RCVTIMEO, usec: 1}, got[4])
	require.Equal(t, timeval{name: syscall.SO_SNDTIMEO, usec: 1}, got[5])
}
