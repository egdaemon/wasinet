//go:build wasip1

package autohijack

import "github.com/egdaemon/wasinet"

func init() {
	wasinet.Hijack()
}
