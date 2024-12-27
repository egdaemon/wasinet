package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"

	"github.com/egdaemon/wasinet/wasinet"
)

func digest(b []byte) string {
	d := md5.Sum(b)
	return hex.EncodeToString(d[:])
}
func testhttpserver() (err error) {

	var (
		l   net.Listener
		buf bytes.Buffer
	)

	_, err = io.CopyN(&buf, rand.Reader, 16*1024)
	if err != nil {
		return err
	}

	m := http.NewServeMux()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.Copy(w, bytes.NewBuffer(buf.Bytes())); err != nil {
			log.Println("copy failed", err)
			return
		}
	})

	if l, err = net.Listen("tcp", ":0"); err != nil {
		return err
	}
	defer l.Close()

	go func() {
		if err = http.Serve(l, m); err != nil {
			log.Println(err)
		}
	}()

	rsp, err := http.Get(fmt.Sprintf("http://%s", l.Addr().String()))
	if err != nil {
		return err
	}
	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d - %s", rsp.StatusCode, rsp.Status)
	}
	received, err := io.ReadAll(rsp.Body)
	if err != nil {
		return err
	}

	if e, a := digest(buf.Bytes()), digest(received); e != a {
		return fmt.Errorf("data doesn't match expected %s vs %s", a, e)
	}

	log.Println("successfully ran http server")
	return nil
}

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	log.SetFlags(log.Flags() | log.Lshortfile)
	wasinet.Hijack()

	ip, err := net.ResolveTCPAddr("tcp", "www.google.com:443")
	if err == nil {
		log.Println("IP ADDRESS", ip.IP, ip.Port)
	} else {
		log.Fatalln("tcp resolution failed", err)
	}
	addresses, err := net.DefaultResolver.LookupIP(context.Background(), "ip", "www.google.com")
	if err == nil {
		log.Println("addresses", addresses)
	} else {
		log.Fatalln("ip resolution failed", err)
	}

	log.Println("http get request")
	rsp, err := http.Get("https://www.google.com")
	if err == nil && rsp.StatusCode == http.StatusOK {
		log.Println("successfully fetched google.com")
	} else {
		log.Fatalln("unable to fetch http", err)
	}

	// if err = testhttpserver(); err != nil {
	// 	log.Println("http server failed", err)
	// }
}
