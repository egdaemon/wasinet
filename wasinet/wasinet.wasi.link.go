//go:build wasip1

package wasinet

import (
	"fmt"
	_ "net"
	_ "unsafe"
)

//go:linkname syscall_derp syscall.fd_fdstat_get_type
func syscall_derp(fd int) (uint8, error)

//go:linkname derp net.fd_fdstat_get_type
func derp(fd int) (uint8, error) {
	fmt.Println("DERP DERP", fd)
	return 0, nil
}
