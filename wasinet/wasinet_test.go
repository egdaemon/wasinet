package wasinet_test

import (
	"context"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/egdaemon/wasinet/wasinet"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	slog.SetLogLoggerLevel(slog.LevelInfo)
	log.SetFlags(log.Flags() | log.Lshortfile)
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{AddSource: true})))
	os.Exit(m.Run())
}

type addrconn interface {
	Addr() net.Addr
}

type udpaddr struct {
	net.PacketConn
}

func (t udpaddr) Addr() net.Addr {
	return t.LocalAddr()
}

func listentcp(t testing.TB, network, address string) net.Listener {
	li, err := wasinet.Listen(context.Background(), network, address)
	require.NoError(t, err)
	go func() {
		for conn, err := li.Accept(); err == nil; conn, err = li.Accept() {
			server, client := net.Pipe()
			go func(c net.Conn) {
				if _, err := io.Copy(c, server); err != nil {
					slog.Error("server copy failed", slog.Any("error", err))
				}
			}(conn)
			go func(c net.Conn) {
				defer c.Close()
				if _, err := io.Copy(client, c); err != nil {
					slog.Error("client copy failed", slog.Any("error", err))
				}
			}(conn)
		}
	}()
	t.Cleanup(func() {
		require.NoError(t, li.Close())
	})

	return li
}

func listenudp(t testing.TB, network, address string) addrconn {
	li, err := wasinet.ListenPacket(context.Background(), network, address)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, li.Close())
	})

	return udpaddr{PacketConn: li}
}

func httpclient() http.Client {
	dialer := &net.Dialer{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial:     wasinet.DialContext,
		},
	}

	return http.Client{
		Transport: &http.Transport{
			DialContext: dialer.DialContext,
		},
	}
}
