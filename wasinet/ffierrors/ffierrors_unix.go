package ffierrors

import (
	"log"
	"syscall"
)

// maps native codes to wasi codes.
func ErrnoTranslate(err syscall.Errno) syscall.Errno {
	switch err {
	case syscall.EINPROGRESS:
		return EINPROGRESS
	default:
		log.Printf("unammped Errno %d - %s\n", int(err), err)
		// unmapped.
		return err
	}
}
