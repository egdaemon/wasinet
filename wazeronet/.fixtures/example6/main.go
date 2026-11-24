package main

import (
	"errors"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/egdaemon/wasinet/wasinet"
)

// dials a server that answers "ping" with "pong" and then neither reads nor writes, and expects
// reads and writes to fail once their deadlines pass, and to wait for the server once cleared.
func main() {
	var (
		timeout net.Error
		chunk   = make([]byte, 64*1024)
		pong    = make([]byte, 4)
	)

	log.SetFlags(log.Flags() | log.Lshortfile)
	wasinet.Hijack()

	conn, err := wasinet.Dial("unix", "/test/socket")
	if err != nil {
		log.Fatalln("unable to dial", err)
	}
	defer conn.Close()

	started := time.Now()
	if err = conn.SetReadDeadline(started.Add(50 * time.Millisecond)); err != nil {
		log.Fatalln("set read deadline failed", err)
	}
	if _, err = conn.Read(pong); !errors.Is(err, os.ErrDeadlineExceeded) || !errors.As(err, &timeout) || !timeout.Timeout() {
		log.Fatalln("read did not time out", err)
	}
	if elapsed := time.Since(started); elapsed < 50*time.Millisecond {
		log.Fatalln("read timed out early", elapsed)
	}

	if err = conn.SetReadDeadline(time.Time{}); err != nil {
		log.Fatalln("clear read deadline failed", err)
	}
	if _, err = conn.Write([]byte("ping")); err != nil {
		log.Fatalln("write failed", err)
	}
	if _, err = io.ReadFull(conn, pong); err != nil {
		log.Fatalln("read failed", err)
	}
	if string(pong) != "pong" {
		log.Fatalf("unexpected reply %q\n", pong)
	}

	if err = conn.SetWriteDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
		log.Fatalln("set write deadline failed", err)
	}
	for err == nil {
		_, err = conn.Write(chunk)
	}
	if !errors.Is(err, os.ErrDeadlineExceeded) {
		log.Fatalln("write did not time out", err)
	}
}
