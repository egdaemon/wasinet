//go:build !wasip1

package wasinetruntime

import (
	"context"
	"net/netip"

	"github.com/egdaemon/wasinetruntime/internal/langx"
	"golang.org/x/sys/unix"
)

//	func Wazero(host wazero.HostModuleBuilder) wazero.HostModuleBuilder {
//		return host.NewFunctionBuilder().
//			WithFunc(wasisocket.Open).Export("socket_open")
//	}

type IP interface {
	Allow(...netip.Prefix) IP
}

type Option func(*network)

func OptionAllow(cidrs ...netip.Prefix) Option {
	return func(s *network) {
		s.allow = append(s.allow, cidrs...)
	}
}

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

func (t network) SendTo(ctx context.Context, fd int, p []byte, flags int, to unix.Sockaddr) error {
	return unix.Sendto(fd, p, flags, to)
}

func (t network) RecvFrom(ctx context.Context, fd int, p []byte, flags int) (int, unix.Sockaddr, error) {
	return unix.Recvfrom(fd, p, flags)
}

func (t network) LocalAddr(ctx context.Context, fd int) (unix.Sockaddr, error) {
	return unix.Getsockname(fd)
}

func (t network) PeerAddr(ctx context.Context, fd int) (unix.Sockaddr, error) {
	return unix.Getpeername(fd)
}
