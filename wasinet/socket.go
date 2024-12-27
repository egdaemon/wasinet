package wasinet

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/egdaemon/wasinet/wasinet/ffi"
	"github.com/egdaemon/wasinet/wasinet/ffierrors"
	"github.com/egdaemon/wasinet/wasinet/internal/errorsx"
)

const (
	oplisten = "listen"
	opdial   = "dial"
)

func lookupAddr(_ context.Context, op, network, address string) ([]net.Addr, error) {
	switch network {
	case "unix", "unixgram":
		return []net.Addr{&net.UnixAddr{Name: address, Net: network}}, nil
	default:
	}

	hostname, service, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	port, err := resolveport(network, service)
	if err != nil {
		return nil, os.NewSyscallError("resolveport", err)
	}

	ips, err := resolveaddrip(op, network, hostname)
	if err != nil {
		return nil, os.NewSyscallError("resolveaddrip", err)
	}

	addrs := make([]net.Addr, 0, len(ips))
	for _, ip := range ips {
		addrs = append(addrs, netaddr(network, ip, port))
	}

	if len(addrs) == 0 {
		return nil, &net.DNSError{
			Err:        "lookup failed",
			Name:       hostname,
			IsNotFound: true,
		}
	}

	return addrs, nil
}

func socket(af, sotype, proto int) (fd int, err error) {
	var newfd int32 = -1
	errno := sock_open(int32(af), int32(sotype), int32(proto), unsafe.Pointer(&newfd))
	if errno != 0 {
		return -1, errno
	}
	return int(newfd), nil
}

func bind(fd int, sa sockaddr) error {
	rsa := sa.sockaddr()
	rawaddr, rawaddrlen := ffi.Pointer(&rsa)
	errno := sock_bind(int32(fd), rawaddr, rawaddrlen)
	runtime.KeepAlive(sa)
	if errno != 0 {
		return errno
	}
	return nil
}

func listen(fd int, backlog int) error {
	if errno := sock_listen(int32(fd), int32(backlog)); errno != 0 {
		return errno
	}
	return nil
}

func connect(fd int, sa sockaddr) error {
	log.Printf("connect %d %+v\n", fd, errorsx.Stack())
	rsa := sa.sockaddr()
	rawaddr, rawaddrlen := ffi.Pointer(&rsa)
	err := ffierrors.Error(sock_connect(int32(fd), rawaddr, rawaddrlen))
	runtime.KeepAlive(sa)
	return err
}

func getsockopt(fd, level, opt int) (value int, err error) {
	var n int32
	errno := ffierrors.Error(sock_getsockopt(int32(fd), uint32(level), uint32(opt), unsafe.Pointer(&n), 4))
	return int(n), errno
}

func setsockopt(fd, level, opt int, value int) error {
	var n = int32(value)
	errno := sock_setsockopt(int32(fd), uint32(level), uint32(opt), unsafe.Pointer(&n), 4)
	if errno != 0 {
		return errno
	}
	return nil
}

func getrawsockname(fd int) (rsa rawsocketaddr, err error) {
	rsaptr, rsalength := ffi.Pointer(&rsa)
	errno := sock_getlocaladdr(int32(fd), rsaptr, rsalength)
	return rsa, ffierrors.Error(errno)
}

func getsockname(fd int) (sa sockaddr, err error) {
	rsa, err := getrawsockname(fd)
	if err != nil {
		return nil, err
	}
	return rawtosockaddr(&rsa)
}

func getrawpeername(fd int) (rsa rawsocketaddr, err error) {
	rsaptr, rsalength := ffi.Pointer(&rsa)
	errno := sock_getpeeraddr(int32(fd), rsaptr, rsalength)
	return rsa, ffierrors.Error(errno)
}

func getpeername(fd int) (sockaddr, error) {
	rsa, err := getrawpeername(fd)
	if err != nil {
		return nil, err
	}
	return rawtosockaddr(&rsa)
}

type sockaddr interface {
	sockaddr() rawsocketaddr
}

type sockipaddr[T any] struct {
	addr T
	port uint32
}

func (s sockipaddr[T]) sockaddr() (raddr rawsocketaddr) {
	ptr, plen := ffi.Pointer(&s)
	buf := errorsx.Must(ffi.SliceRead[byte](ffi.Native{}, ptr, plen))

	switch x := any(s.addr).(type) {
	case sockip4:
		raddr.family = uint16(sock_determine_host_af_family(syscall.AF_INET))
	case sockip6:
		raddr.family = uint16(sock_determine_host_af_family(syscall.AF_INET6))
	default:
		log.Printf("unknown socket type %T - default famimly to AF_INET\n", x)
		raddr.family = uint16(sock_determine_host_af_family(syscall.AF_INET))
	}

	copy(raddr.addr[:], buf)
	return raddr
}

type sockip4 struct {
	ip [4]byte
}

type sockip6 struct {
	ip   [16]byte
	zone string
}

type sockaddrUnix struct {
	name string
}

func (s *sockaddrUnix) sockaddr() rawsocketaddr {
	ptr, plen := ffi.Pointer(&s)
	buf := errorsx.Must(ffi.SliceRead[byte](ffi.Native{}, ptr, plen))

	raddr := rawsocketaddr{
		family: uint16(sock_determine_host_af_family(syscall.AF_UNIX)),
	}
	copy(raddr.addr[:], buf)
	return raddr
}

type rawsocketaddr struct {
	family uint16
	addr   [126]byte
}

func recvfrom(fd int, iovs [][]byte, flags int32) (n int, addr rawsocketaddr, oflags int32, err error) {
	vecs := ffi.VectorSlice(iovs...)
	iovsptr, iovslen := ffi.Slice(vecs)
	addrptr, addrlen := ffi.Pointer(&addr)

	errno := sock_recv_from(
		int32(fd),
		iovsptr, iovslen,
		addrptr, addrlen,
		flags,
		unsafe.Pointer(&n),
		unsafe.Pointer(&oflags),
	)
	runtime.KeepAlive(addrptr)
	runtime.KeepAlive(iovsptr)
	runtime.KeepAlive(iovs)
	return n, addr, oflags, ffierrors.Error(errno)
}

func sendto(fd int, iovs [][]byte, addr rawsocketaddr, flags int32) (int, error) {
	vecs := ffi.VectorSlice(iovs...)
	iovsptr, iovslen := ffi.Slice(vecs)
	addrptr, addrlen := ffi.Pointer(&addr)

	nwritten := int(0)
	errno := sock_send_to(
		int32(fd),
		iovsptr, iovslen,
		addrptr, addrlen,
		flags,
		unsafe.Pointer(&nwritten),
	)
	runtime.KeepAlive(addr)
	runtime.KeepAlive(iovs)
	return nwritten, ffierrors.Error(errno)
}

func strlen(b []byte) (n int) {
	for n < len(b) && b[n] != 0 {
		n++
	}
	return n
}

func networkip(network string) string {
	switch network {
	case "tcp", "udp":
		return "ip"
	case "tcp4", "udp4":
		return "ip4"
	case "tcp6", "udp6":
		return "ip6"
	default:
		return ""
	}
}

func netaddr(network string, ip net.IP, port int) net.Addr {
	switch network {
	case "tcp", "tcp4", "tcp6":
		return &net.TCPAddr{IP: ip, Port: port}
	case "udp", "udp4", "udp6":
		return &net.UDPAddr{IP: ip, Port: port}
	}
	return nil
}

func netOpErr(op string, addr net.Addr, err error) error {
	if err == nil {
		return nil
	}

	return &net.OpError{
		Op:   op,
		Net:  addr.Network(),
		Addr: addr,
		Err:  err,
	}
}

func unresolvedaddr(network, address string) net.Addr {
	return &unresolvedaddress{network: network, address: address}
}

type unresolvedaddress struct{ network, address string }

func (na *unresolvedaddress) Network() string { return na.network }
func (na *unresolvedaddress) String() string  { return na.address }

func setNonBlock(fd int) error {
	// if err := syscall.SetNonblock(fd, true); err != nil {
	// 	return os.NewSyscallError("setnonblock", err)
	// }
	return nil
}

func netaddrToSockaddr(addr net.Addr) (sockaddr, error) {
	ipaddr := func(ip net.IP, zone string, port int) (sockaddr, error) {
		if ipv4 := ip.To4(); ipv4 != nil {
			return sockipaddr[sockip4]{addr: sockip4{ip: ([4]byte)(ipv4)}, port: uint32(port)}, nil
		} else if len(ip) == net.IPv6len {
			return sockipaddr[sockip6]{addr: sockip6{ip: ([16]byte)(ip), zone: zone}, port: uint32(port)}, nil
		} else {
			return nil, &net.AddrError{
				Err:  "unsupported address type",
				Addr: addr.String(),
			}
		}
	}

	switch a := addr.(type) {
	case *net.IPAddr:
		return ipaddr(a.IP, a.Zone, 0)
	case *net.TCPAddr:
		return ipaddr(a.IP, a.Zone, a.Port)
	case *net.UDPAddr:
		return ipaddr(a.IP, a.Zone, a.Port)
	case *net.UnixAddr:
		return &sockaddrUnix{name: a.Name}, nil
	}

	return nil, &net.AddrError{
		Err:  "unsupported address type",
		Addr: addr.String(),
	}
}

func netaddrproto(_ net.Addr) int {
	return syscall.IPPROTO_IP
}

func socketType(addr net.Addr) (int, error) {
	switch addr.Network() {
	case "tcp", "tcp4", "tcp6", "unix", "unixpacket":
		return syscall.SOCK_STREAM, nil
	case "udp", "udp4", "udp6", "unixgram":
		return syscall.SOCK_DGRAM, nil
	default:
		return -1, syscall.EPROTOTYPE
	}
}

func setReuseAddress(fd int) error {
	if err := setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, 1); err != nil {
		// The runtime may not support the option; if that's the case and the
		// address is already in use, binding the socket will fail and we will
		// report the error then.
		switch {
		case errors.Is(err, syscall.ENOPROTOOPT):
		case errors.Is(err, syscall.EINVAL):
		default:
			return os.NewSyscallError("setsockopt", err)
		}
	}
	return nil
}

type unixConn struct {
	net.Conn
}

func (c *unixConn) CloseRead() error {
	if cr, ok := c.Conn.(closeReader); ok {
		return cr.CloseRead()
	}

	return &net.OpError{
		Op:     "close",
		Net:    "unix",
		Source: c.LocalAddr(),
		Err:    syscall.ENOTSUP,
	}
}

func (c *unixConn) CloseWrite() error {
	if cw, ok := c.Conn.(closeWriter); ok {
		return cw.CloseWrite()
	}
	return &net.OpError{
		Op:     "close",
		Net:    "unix",
		Source: c.LocalAddr(),
		Err:    syscall.ENOTSUP,
	}
}

type closeReader interface {
	CloseRead() error
}

type closeWriter interface {
	CloseWrite() error
}

const (
	readSyscallName     = "read"
	readFromSyscallName = "recvfrom"
	readMsgSyscallName  = "recvmsg"
	writeSyscallName    = "write"
	writeToSyscallName  = "sendto"
	writeMsgSyscallName = "sendmsg"
)

func sockipaddrToUnix(sa sockaddr) net.Addr {
	if sa == nil {
		return &net.UnixAddr{}
	}
	switch proto := sa.(type) {
	case *sockipaddr[sockaddrUnix]:
		return &net.UnixAddr{Name: proto.addr.name, Net: "unix"}
	default:
		return nil
	}
}

func sockipaddrToUnixgram(sa sockaddr) net.Addr {
	if sa == nil {
		return &net.UnixAddr{}
	}
	switch proto := sa.(type) {
	case *sockipaddr[sockaddrUnix]:
		return &net.UnixAddr{Name: proto.addr.name, Net: "unixgram"}
	default:
		return nil
	}
}

func sockipaddrToUnixpacket(sa sockaddr) net.Addr {
	if sa == nil {
		return &net.UnixAddr{}
	}
	switch proto := sa.(type) {
	case *sockipaddr[sockaddrUnix]:
		return &net.UnixAddr{Name: proto.addr.name, Net: "unixpacket"}
	}
	return nil
}
func sockipaddrToTCP(sa sockaddr) net.Addr {
	if sa == nil {
		return &net.TCPAddr{}
	}

	switch proto := sa.(type) {
	case *sockipaddr[sockip4]:
		return &net.TCPAddr{IP: proto.addr.ip[0:], Port: int(proto.port)}
	case *sockipaddr[sockip6]:
		return &net.TCPAddr{IP: proto.addr.ip[0:], Port: int(proto.port), Zone: ""}
	}
	return nil
}

func sockipaddrToUDP(sa sockaddr) net.Addr {
	if sa == nil {
		return &net.UDPAddr{}
	}

	switch proto := sa.(type) {
	case *sockipaddr[sockip4]:
		return &net.UDPAddr{IP: proto.addr.ip[0:], Port: int(proto.port)}
	case *sockipaddr[sockip6]:
		return &net.UDPAddr{IP: proto.addr.ip[0:], Port: int(proto.port), Zone: ""}
	default:
		return nil
	}
}

func sockipaddrToIP(sa sockaddr) net.Addr {
	if sa == nil {
		return &net.IPAddr{}
	}

	switch proto := sa.(type) {
	case *sockipaddr[sockip4]:
		return &net.IPAddr{IP: proto.addr.ip[0:]}
	case *sockipaddr[sockip6]:
		return &net.IPAddr{IP: proto.addr.ip[0:], Zone: ""}
	default:
		return nil
	}
}

func socnetwork(family, sotype int) string {
	switch family {
	case syscall.AF_INET, syscall.AF_INET6:
		switch sotype {
		case syscall.SOCK_STREAM:
			return "tcp"
		case syscall.SOCK_DGRAM:
			return "udp"
		case syscall.SOCK_RAW:
			return "ip"
		}
	case syscall.AF_UNIX:
		switch sotype {
		case syscall.SOCK_STREAM:
			return "unix"
		case syscall.SOCK_DGRAM:
			return "unixgram"
		case syscall.SOCK_SEQPACKET:
			return "unixpacket"
		}
	}

	return ""
}

type _AFFamilyMap struct {
	AF_INET  int32
	AF_INET6 int32
	AF_UNIX  int32
}

var (
	mapped = _AFFamilyMap{}
)

func init() {
	mapped.AF_UNIX = sock_determine_host_af_family(syscall.AF_UNIX)
	mapped.AF_INET = sock_determine_host_af_family(syscall.AF_INET)
	mapped.AF_INET6 = sock_determine_host_af_family(syscall.AF_INET6)
}

func sockipToNetAddr(family, sotype int) func(sa sockaddr) net.Addr {
	switch int32(family) {
	case mapped.AF_INET, mapped.AF_INET6:
		switch sotype {
		case syscall.SOCK_STREAM:
			return sockipaddrToTCP
		case syscall.SOCK_DGRAM:
			return sockipaddrToUDP
		case syscall.SOCK_RAW:
			return sockipaddrToIP
		}
	case mapped.AF_UNIX:
		switch sotype {
		case syscall.SOCK_STREAM:
			return sockipaddrToUnix
		case syscall.SOCK_DGRAM:
			return sockipaddrToUnixgram
		case syscall.SOCK_SEQPACKET:
			return sockipaddrToUnixpacket
		}
	}
	log.Println(family, mapped.AF_INET, mapped.AF_INET6, mapped.AF_UNIX, "|", sotype, syscall.SOCK_STREAM, syscall.SOCK_DGRAM)
	return func(sa sockaddr) net.Addr { return nil }
}

func newFD(sysfd int, family int, sotype int, net string, laddr, raddr net.Addr) *netFD {
	if laddr == nil {
		laddr = sockipToNetAddr(family, sotype)(nil)
	}

	if raddr == nil {
		raddr = sockipToNetAddr(family, sotype)(nil)
	}
	log.Println("newFD", family, sotype, laddr, raddr)
	s := &netFD{
		fd:     sysfd,
		family: family,
		sotype: sotype,
		net:    net,
		laddr:  laddr,
		raddr:  raddr,
		rraddr: new(rawsocketaddr),
	}
	runtime.SetFinalizer(s, (*netFD).Close)
	return s
}

type netsysconn struct {
	*netFD
}

func (t netsysconn) Control(f func(fd uintptr)) error {
	f(uintptr(t.fd))
	return nil
}

func (t netsysconn) Read(f func(fd uintptr) (done bool)) error {
	for !f(uintptr(t.fd)) {
		// Tthere are almost certainly bugs here not checking to ensure its open for example.
	}
	return nil
}

// Write is like Read but for writing.
func (t netsysconn) Write(f func(fd uintptr) (done bool)) error {
	for !f(uintptr(t.fd)) {
		// Tthere are almost certainly bugs here not checking to ensure its open for example.
	}
	return nil
}

var _ syscall.RawConn = netsysconn{}

type netFD struct {
	fd   int
	dead *int
	// immutable until Close
	family int
	sotype int
	net    string
	laddr  net.Addr
	raddr  net.Addr
	rraddr *rawsocketaddr
}

func (c *netFD) discard() { c.dead = &c.fd }

func (c *netFD) ok() bool { return c != nil && c.dead == nil }

func (fd *netFD) SyscallConn() (syscall.RawConn, error) {
	return netsysconn{fd}, nil
}

func (fd *netFD) Close() error {
	if !fd.ok() {
		return nil
	}
	defer fd.discard()
	return fd.shutdown(syscall.SHUT_RDWR)
}

func (fd *netFD) shutdown(how int) error {
	if !fd.ok() {
		return nil
	}
	err := sock_shutdown(int32(fd.fd), int32(how))
	switch err {
	case syscall.ENOTCONN:
		switch fd.laddr.(type) {
		case *net.UDPAddr:
			err = ffierrors.ErrnoSuccess()
		default:
		}
	}

	runtime.SetFinalizer(fd, nil)
	runtime.KeepAlive(fd)
	return wrapSyscallError("shutdown", ffierrors.Error(err))
}

func (fd *netFD) closeRead() error {
	return fd.shutdown(syscall.SHUT_RD)
}

func (fd *netFD) closeWrite() error {
	return fd.shutdown(syscall.SHUT_WR)
}

func (fd *netFD) Read(p []byte) (n int, err error) {
	n, _, _, err = recvfrom(fd.fd, [][]byte{p}, 0)
	return n, wrapSyscallError(readSyscallName, err)
}

func (fd *netFD) initremote() error {
	var (
		saddr sockaddr
	)
	// log.Println("initremote initiated", fd.raddr, fmt.Sprintf("%+v", errorsx.Stack()))
	if fd.raddr == nil {
		return fmt.Errorf("no remote address")
	}

	saddr, err := netaddrToSockaddr(fd.raddr)
	if err != nil {
		return err
	}

	*fd.rraddr = saddr.sockaddr()
	return nil
}

func (fd *netFD) Write(p []byte) (n int, err error) {
	n, err = sendto(fd.fd, [][]byte{p}, *fd.rraddr, 0)
	return n, wrapSyscallError(writeSyscallName, err)
}

func (c *netFD) SetDeadline(t time.Time) error {
	return nil // TODO
}

func (c *netFD) SetReadDeadline(t time.Time) error {
	return nil // TODO
}

func (c *netFD) SetWriteDeadline(t time.Time) error {
	return nil // TODO
}

func (fd *netFD) LocalAddr() net.Addr {
	return fd.laddr
}

func (fd *netFD) RemoteAddr() net.Addr {
	return fd.raddr
}

// wrapSyscallError takes an error and a syscall name. If the error is
// a syscall.Errno, it wraps it in an os.SyscallError using the syscall name.
func wrapSyscallError(name string, err error) error {
	if _, ok := err.(syscall.Errno); ok {
		err = os.NewSyscallError(name, err)
	}
	return err
}
