package wasip1net

import (
	"net"
	"os"
)

func Socket(fd uintptr) *os.File {
	return newFile(int(fd), "")
}

func Conn(f *os.File) (c net.Conn, err error) {
	return newConn(f)
}
