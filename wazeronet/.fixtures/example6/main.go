package main

import (
	"log"
	"os"
	"time"

	"github.com/egdaemon/wasinet/wasinet"
)

// exercises SO_RCVTIMEO/SO_SNDTIMEO through the deadline methods on a connection so the host
// can assert the timevals that arrive over the wasi boundary. the host test knows this order.
func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	wasinet.Hijack()

	addr := os.Getenv("DIAL_ADDR")
	if addr == "" {
		log.Fatalln("DIAL_ADDR not set")
	}

	conn, err := wasinet.Dial("tcp", addr)
	if err != nil {
		log.Fatalln("dial failed", err)
	}
	defer conn.Close()

	steps := []struct {
		name string
		do   func() error
	}{
		{"read deadline in 1.5s", func() error { return conn.SetReadDeadline(time.Now().Add(1500 * time.Millisecond)) }},
		{"write deadline in 1.5s", func() error { return conn.SetWriteDeadline(time.Now().Add(1500 * time.Millisecond)) }},
		{"read deadline cleared", func() error { return conn.SetReadDeadline(time.Time{}) }},
		{"write deadline cleared", func() error { return conn.SetWriteDeadline(time.Time{}) }},
		{"read deadline in the past", func() error { return conn.SetReadDeadline(time.Now().Add(-time.Second)) }},
		{"write deadline in the past", func() error { return conn.SetWriteDeadline(time.Now().Add(-time.Second)) }},
	}

	for _, s := range steps {
		if err := s.do(); err != nil {
			log.Fatalln(s.name, "failed", err)
		}
	}

	// a datagram socket has a different socket type than the stream one above, which is what
	// distinguishes passing SOL_SOCKET as the level from passing the socket type by mistake.
	pc, err := wasinet.Dial("udp", addr)
	if err != nil {
		log.Fatalln("dial udp failed", err)
	}
	defer pc.Close()

	if err := pc.SetReadDeadline(time.Now().Add(1500 * time.Millisecond)); err != nil {
		log.Fatalln("udp read deadline failed", err)
	}

	log.Println("deadlines set on", addr)
}
