package wasinet

import "net"

// hijack the net.DefaultResolver
func Hijack() {
	net.DefaultResolver.Dial = DialContext
}
