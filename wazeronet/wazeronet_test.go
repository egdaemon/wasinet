package wazeronet_test

import (
	"context"
	"crypto/rand"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/egdaemon/wasinet/wasinet/testx"
	"github.com/egdaemon/wasinet/wasinet/wnetruntime"
	"github.com/egdaemon/wasinet/wazeronet"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

func TestMain(m *testing.M) {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	// slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{AddSource: true})))
	// log.SetFlags(log.Lshortfile)
	// log.SetOutput(io.Discard)
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

func compileAndRun(ctx context.Context, t *testing.T, path string) error {
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
	wasienv, err := wasi_snapshot_preview1.NewBuilder(runtime).Instantiate(ctx)
	if err != nil {
		return err
	}
	defer wasienv.Close(ctx)

	wasinet, err := wazeronet.Module(runtime, wnetruntime.Unrestricted()).Instantiate(ctx)
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

	require.NoError(t, compileAndRun(ctx, t, testx.Fixture("example1", "main.go")))
}
