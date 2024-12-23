package wasinet_test

import (
	"log"
	"net"
	"os"
	"testing"

	"github.com/egdaemon/wasinetruntime/wasinet"
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
	li, err := net.Listen(network, address)
	if err != nil {
		log.Println("checkpoint")
		t.Fatal(err)
	}

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
