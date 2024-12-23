package wasinet_test

import (
	"io"
	"log"
	"net"
	"os"
	"testing"

	"github.com/egdaemon/wasinet/wasinet"
)

func TestMain(m *testing.M) {
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
	if err != nil {
		log.Println("checkpoint")
		t.Fatal(err)
	}

	go func() {
		for conn, err := li.Accept(); err == nil; conn, err = li.Accept() {
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
	t.Cleanup(func() {
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
