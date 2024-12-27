package wasinet_test

import (
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
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{AddSource: true})))
	log.SetFlags(log.Lshortfile)
	log.SetOutput(os.Stderr)
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
	li, err := wasinet.Listen(network, address)
	log.Println("checkpoint")
	require.NoError(t, err)
	log.Println("checkpoint")
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
		log.Println("DERP DERP")
		if err := li.Close(); err != nil {
			t.Fatal(err)
		}
	})

	return li
}

func listenudp(t testing.TB, network, address string) addrconn {
	li, err := wasinet.ListenPacket(network, address)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := li.Close(); err != nil {
			t.Fatal(err)
		}
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
