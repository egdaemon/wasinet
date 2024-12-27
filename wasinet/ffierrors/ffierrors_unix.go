package ffierrors

import "syscall"

// maps native codes to wasi codes.
func ErrnoTranslate(err syscall.Errno) syscall.Errno {
	switch err {
	case syscall.EINPROGRESS:
		return EINPROGRESS
	default:
		// unmapped.
		return err
	}
}
