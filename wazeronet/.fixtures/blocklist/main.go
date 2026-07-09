package main

import (
	"bytes"
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
	wasinet.Hijack()

	addr := os.Getenv("DIAL_ADDR")
	if addr == "" {
		log.Fatalln("DIAL_ADDR not set")
	}

	if err := checkTransfer("tcp", addr, 1024); err != nil {
		log.Fatalln("transfer test failed", err)
	}

	log.Println("successfully connected to", addr)
}

func checkTransfer(network, addr string, amount int64) error {
	var (
		serr       error
		amountsent int64
	)

	conn, err := wasinet.Dial(network, addr)
	if err != nil {
		return err
	}
	defer conn.Close()

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
