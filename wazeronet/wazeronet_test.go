package wazeronet_test

import (
	"context"
	"crypto/rand"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/egdaemon/wasinet/wasinet/testx"
	"github.com/egdaemon/wasinet/wasinet/wnetruntime"
	"github.com/egdaemon/wasinet/wazeronet"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/experimental/logging"
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
	ctx = experimental.WithFunctionListenerFactory(ctx,
		logging.NewHostLoggingListenerFactory(os.Stderr, logging.LogScopeAll))

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

func TestUnixExample1(t *testing.T) {
	ctx, done := testx.WithDeadline(t)
	defer done()

	ctx, done = context.WithTimeout(ctx, 5*time.Second)
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
		)
	}))
}
