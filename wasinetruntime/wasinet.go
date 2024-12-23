//go:build !wasip1

package wasinetruntime

import (
	"context"
	"net/netip"

	"github.com/egdaemon/wasinet/internal/langx"
	"golang.org/x/sys/unix"
)

type IP interface {
	Allow(...netip.Prefix) IP
}

type Option func(*network)

func OptionAllow(cidrs ...netip.Prefix) Option {
	return func(s *network) {
		s.allow = append(s.allow, cidrs...)
	}
}

// unrestricted network defaults.
func Unrestricted(opts ...Option) *network {
	return langx.Autoptr(
		langx.Clone(
			network{},
			opts...,
		),
	)
}

// the network by default disallows all network activity. use unrestricted
// or manually configure using options.
func New(opts ...Option) *network {
	return langx.Autoptr(langx.Clone(network{}, opts...))
}

type network struct {
	allow []netip.Prefix
}

func (t network) Open(ctx context.Context, domain, utype, protocol int) (int, error) {
	return unix.Socket(domain, utype, protocol)
}

func (t network) Bind(ctx context.Context, fd int, sa unix.Sockaddr) error {
	return unix.Bind(fd, sa)
}

func (t network) Connect(ctx context.Context, fd int, sa unix.Sockaddr) error {
	return unix.Connect(fd, sa)
}

func (t network) Listen(ctx context.Context, fd, backlog int) error {
	return unix.Listen(fd, backlog)
}

func (t network) LocalAddr(ctx context.Context, fd int) (unix.Sockaddr, error) {
	return unix.Getsockname(fd)
}

func (t network) PeerAddr(ctx context.Context, fd int) (unix.Sockaddr, error) {
	return unix.Getpeername(fd)
}

func (t network) SetSocketOption() {

}

func (t network) GetSocketOption() {

}

func (t network) Shutdown() {

}

func (t network) AddrIP()   {}
func (t network) AddrPort() {}
