package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/egdaemon/wasinet/wasinet"
)

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	var (
		err error
	)
	wasinet.Hijack()

	if _, err := os.Stat("/test/socket"); err != nil {
		log.Fatalln("unable to stat socket path", err)
	}

	if err = checkTransfer(context.Background(), "unix", "/test/socket", 1024); err != nil {
		log.Fatalln("transfer test failed", err)
	}
}

func checkTransfer(ctx context.Context, network, path string, amount int64) error {
	var (
		serr       error
		amountsent int64
	)

	// need to use wasinet.Dial because unix sockets do not pass through
	// the net resolver code path.
	conn, err := wasinet.Dial(network, path)
	if err != nil {
		return err
	}

	digestsent := md5.New()
	digestrecv := md5.New()

	go func() {
		amountsent, serr = io.CopyN(conn, io.TeeReader(rand.Reader, digestsent), amount)
	}()

	n, err := io.Copy(digestrecv, io.LimitReader(conn, amount))
	if err != nil {
		return err
	}

	if serr != nil {
		return serr
	}

	if amount != n {
		return fmt.Errorf("didnt receive all data %d != %d", amount, n)
	}

	if amount != amountsent {
		return fmt.Errorf("didnt receive all data %d != %d", amount, amountsent)
	}

	if !bytes.Equal(digestsent.Sum(nil), digestrecv.Sum(nil)) {
		return fmt.Errorf("digests didnt match")
	}

	return nil
}
