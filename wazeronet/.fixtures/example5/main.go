package main

import (
	"io"
	"log"

	"github.com/egdaemon/wasinet/wasinet"
)

// dials an echo server, half closes the connection after writing, and expects the echo followed by EOF.
func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	wasinet.Hijack()

	conn, err := wasinet.Dial("unix", "/test/socket")
	if err != nil {
		log.Fatalln("unable to dial", err)
	}
	defer conn.Close()

	if _, err = conn.Write([]byte("hello")); err != nil {
		log.Fatalln("write failed", err)
	}

	if err = conn.(interface{ CloseWrite() error }).CloseWrite(); err != nil {
		log.Fatalln("close write failed", err)
	}

	echoed, err := io.ReadAll(conn)
	if err != nil {
		log.Fatalln("read failed", err)
	}

	if string(echoed) != "hello" {
		log.Fatalf("unexpected echo %q\n", echoed)
	}
}
