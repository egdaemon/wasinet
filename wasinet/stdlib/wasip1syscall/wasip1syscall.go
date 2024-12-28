package wasip1syscall

import (
	"log"
	"net"
	"net/netip"
	"syscall"
	"unsafe"

	"github.com/egdaemon/wasinet/wasinet/ffi"
	"github.com/egdaemon/wasinet/wasinet/internal/errorsx"
	"github.com/egdaemon/wasinet/wasinet/internal/langx"
)

const (
	oplisten = "listen"
	opdial   = "dial"
)

// mapping to host's AFFamily values
type _AFFamilyMap struct {
	INET  int32
	INET6 int32
	UNIX  int32
}

var (
	afmap = _AFFamilyMap{}
)

func init() {
	afmap.UNIX = sock_determine_host_af_family(syscall.AF_UNIX)
	afmap.INET = sock_determine_host_af_family(syscall.AF_INET)
	afmap.INET6 = sock_determine_host_af_family(syscall.AF_INET6)
}

func AF() _AFFamilyMap {
	return afmap
}

type RawSocketAddress struct {
	Family  uint16
	Soctype uint16
	Addr    [126]byte
}

type sockaddr interface {
	Sockaddr() RawSocketAddress
}

type addressany[T any] struct {
	family  uint16
	soctype uint16
	addr    T
}

func (s addressany[T]) Sockaddr() (raddr RawSocketAddress) {
	ptr, plen := ffi.Pointer(&s)
	buf := errorsx.Must(ffi.SliceRead[byte](ffi.Native{}, ptr, plen))
	raddr.Family = s.family
	raddr.Soctype = s.soctype
	copy(raddr.Addr[:], buf)
	return raddr
}

type addrip4 struct {
	port uint32
	ip   [4]byte
}

type addrip6 struct {
	port uint32
	ip   [16]byte
	zone string
}

type addrunix struct {
	name string
}

func rawtosockaddr(rsa *RawSocketAddress) (sockaddr, error) {
	switch rsa.Family {
	case syscall.AF_INET:
		addr := (*addressany[addrip4])(unsafe.Pointer(&rsa.Addr))
		return addr, nil
	case syscall.AF_INET6:
		addr := (*addressany[addrip6])(unsafe.Pointer(&rsa.Addr))
		return addr, nil
	case syscall.AF_UNIX:
		addr := (*addressany[addrunix])(unsafe.Pointer(&rsa.Addr))
		return addr, nil
	default:
		return nil, syscall.ENOTSUP
	}
}

func NetaddrToRaw(addr net.Addr) (*RawSocketAddress, error) {
	ipaddr := func(ip net.IP, zone string, port int) (*RawSocketAddress, error) {
		if ipv4 := ip.To4(); ipv4 != nil {
			return langx.Autoptr(addressany[addrip4]{addr: addrip4{ip: ([4]byte)(ipv4), port: uint32(port)}}.Sockaddr()), nil
		} else if len(ip) == net.IPv6len {
			return langx.Autoptr(addressany[addrip6]{addr: addrip6{ip: ([16]byte)(ip), zone: zone, port: uint32(port)}}.Sockaddr()), nil
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
		return langx.Autoptr(addressany[addrunix]{addr: addrunix{name: a.Name}}.Sockaddr()), nil
	}

	return nil, &net.AddrError{
		Err:  "unsupported address type",
		Addr: addr.String(),
	}
}

func NetUnixToRaw(sa *net.UnixAddr) (zero RawSocketAddress) {
	name := sa.Name
	if len(name) == 0 {
		// For consistency across platforms, replace empty unix socket
		// addresses with @. On Linux, addresses where the first byte is
		// a null byte are considered abstract unix sockets, and the first
		// byte is replaced with @.
		name = "@"
	}
	return (&addressany[addrunix]{family: 99, soctype: 99, addr: addrunix{name: name}}).Sockaddr()
}

func NetUnix(v RawSocketAddress) (addrPort *net.UnixAddr, err error) {
	sockaddr, err := rawtosockaddr(&v)
	if err != nil {
		return addrPort, err
	}

	switch unknown := sockaddr.(type) {
	case *addressany[addrunix]:
		return &net.UnixAddr{Name: unknown.addr.name}, nil
	default:
		log.Printf("unsupported address %T\n", unknown)
		return addrPort, syscall.EINVAL
	}
}

func NetipAddrPortToRaw(nap netip.AddrPort) RawSocketAddress {
	if nap.Addr().Is4() || nap.Addr().Is4In6() {
		a := addressany[addrip4]{family: 99, soctype: 99, addr: addrip4{port: uint32(nap.Port()), ip: nap.Addr().As4()}}
		return a.Sockaddr()
	} else {
		a := addressany[addrip6]{family: 99, soctype: 99, addr: addrip6{port: uint32(nap.Port()), ip: nap.Addr().As16()}}
		return a.Sockaddr()
	}
}

func Netipaddrport(v RawSocketAddress) (addrPort netip.AddrPort, err error) {
	sockaddr, err := rawtosockaddr(&v)
	if err != nil {
		return addrPort, err
	}

	switch unknown := sockaddr.(type) {
	case *addressany[addrip4]:
		addrPort = netip.AddrPortFrom(netip.AddrFrom4(unknown.addr.ip), uint16(unknown.addr.port))
	case *addressany[addrip6]:
		addrPort = netip.AddrPortFrom(netip.AddrFrom16(unknown.addr.ip), uint16(unknown.addr.port))
	default:
		log.Printf("unsupported address %T\n", unknown)
		return addrPort, syscall.EINVAL
	}

	return addrPort, nil
}

func NetaddrAFFamily(addr net.Addr) int {
	translated := func(v int32) int {
		return int(sock_determine_host_af_family(v))
	}
	ipfamily := func(ip net.IP) int {
		if ip.To4() == nil {
			return translated(syscall.AF_INET6)
		}

		return translated(syscall.AF_INET)
	}

	switch a := addr.(type) {
	case *net.IPAddr:
		return ipfamily(a.IP)
	case *net.TCPAddr:
		return ipfamily(a.IP)
	case *net.UDPAddr:
		return ipfamily(a.IP)
	case *net.UnixAddr:
		return translated(syscall.AF_UNIX)
	}

	return translated(syscall.AF_INET)
}
