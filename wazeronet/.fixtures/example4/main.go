package main

import (
	"context"
	"io"
	"log"

	"github.com/egdaemon/wasinet/wasinet"
)

// listens on a unix socket under a mounted directory and echoes 5 bytes of the first connection back to its peer.
func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	wasinet.Hijack()

	li, err := wasinet.Listen(context.Background(), "unix", "/test/listen.socket")
	if err != nil {
		log.Fatalln("unable to listen", err)
	}
	defer li.Close()

	conn, err := li.Accept()
	if err != nil {
		log.Fatalln("unable to accept", err)
	}
	defer conn.Close()

	if _, err = io.CopyN(conn, conn, 5); err != nil {
		log.Fatalln("echo failed", err)
	}
}
