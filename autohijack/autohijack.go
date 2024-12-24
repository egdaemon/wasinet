//go:build wasip1

package autohijack

import "github.com/egdaemon/wasinet/wasinet"

func init() {
	wasinet.Hijack()
}
