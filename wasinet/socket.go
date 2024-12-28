package wasinet

import (
	"net"
	"syscall"
)

const (
	oplisten = "listen"
	opdial   = "dial"
)

// func lookupAddr(_ context.Context, op, network, address string) ([]net.Addr, error) {
// 	switch network {
// 	case "unix", "unixgram":
// 		return []net.Addr{&net.UnixAddr{Name: address, Net: network}}, nil
// 	default:
// 	}

// 	hostname, service, err := net.SplitHostPort(address)
// 	if err != nil {
// 		return nil, err
// 	}

// 	port, err := resolveport(network, service)
// 	if err != nil {
// 		return nil, os.NewSyscallError("resolveport", err)
// 	}

// 	ips, err := resolveaddrip(op, network, hostname)
// 	if err != nil {
// 		return nil, os.NewSyscallError("resolveaddrip", err)
// 	}

// 	addrs := make([]net.Addr, 0, len(ips))
// 	for _, ip := range ips {
// 		addrs = append(addrs, netaddr(network, ip, port))
// 	}

// 	if len(addrs) == 0 {
// 		return nil, &net.DNSError{
// 			Err:        "lookup failed",
// 			Name:       hostname,
// 			IsNotFound: true,
// 		}
// 	}

// 	return addrs, nil
// }

// func socket(af, sotype, proto int) (fd int, err error) {
// 	var newfd int32 = -1
// 	errno := sock_open(int32(af), int32(sotype), int32(proto), unsafe.Pointer(&newfd))
// 	if errno != 0 {
// 		return -1, errno
// 	}
// 	return int(newfd), nil
// }

// func bind(fd int, sa sockaddr) error {
// 	rsa := sa.sockaddr()
// 	rawaddr, rawaddrlen := ffi.Pointer(&rsa)
// 	errno := sock_bind(int32(fd), rawaddr, rawaddrlen)
// 	runtime.KeepAlive(sa)
// 	if errno != 0 {
// 		return errno
// 	}
// 	return nil
// }

// func listen(fd int, backlog int) error {
// 	if errno := sock_listen(int32(fd), int32(backlog)); errno != 0 {
// 		return errno
// 	}
// 	return nil
// }

// func connect(fd int, sa sockaddr) error {
// 	rsa := sa.sockaddr()
// 	rawaddr, rawaddrlen := ffi.Pointer(&rsa)
// 	err := ffierrors.Error(sock_connect(int32(fd), rawaddr, rawaddrlen))
// 	runtime.KeepAlive(sa)
// 	return err
// }

// func getsockopt(fd, level, opt int) (value int, err error) {
// 	var n int32
// 	errno := ffierrors.Error(sock_getsockopt(int32(fd), uint32(level), uint32(opt), unsafe.Pointer(&n), 4))
// 	return int(n), errno
// }

// func setsockopt(fd, level, opt int, value int) error {
// 	var n = int32(value)
// 	errno := sock_setsockopt(int32(fd), uint32(level), uint32(opt), unsafe.Pointer(&n), 4)
// 	if errno != 0 {
// 		return errno
// 	}
// 	return nil
// }

// func getrawsockname(fd int) (rsa rawsocketaddr, err error) {
// 	rsaptr, rsalength := ffi.Pointer(&rsa)
// 	errno := ffierrors.Error(sock_getlocaladdr(int32(fd), rsaptr, rsalength))
// 	return rsa, errno
// }

// func getsockname(fd int) (sa sockaddr, err error) {
// 	rsa, err := getrawsockname(fd)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return rawtosockaddr(&rsa)
// }

// func getrawpeername(fd int) (rsa rawsocketaddr, err error) {
// 	rsaptr, rsalength := ffi.Pointer(&rsa)
// 	errno := sock_getpeeraddr(int32(fd), rsaptr, rsalength)
// 	return rsa, ffierrors.Error(errno)
// }

// func getpeername(fd int) (sockaddr, error) {
// 	rsa, err := getrawpeername(fd)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return rawtosockaddr(&rsa)
// }

// type sockaddr interface {
// 	sockaddr() rawsocketaddr
// }

// func netaddrToSockaddr(addr net.Addr) (sockaddr, error) {
// 	ipaddr := func(ip net.IP, zone string, port int) (sockaddr, error) {
// 		if ipv4 := ip.To4(); ipv4 != nil {
// 			return sockipaddr[sockip4]{addr: sockip4{ip: ([4]byte)(ipv4)}, port: uint32(port)}, nil
// 		} else if len(ip) == net.IPv6len {
// 			return sockipaddr[sockip6]{addr: sockip6{ip: ([16]byte)(ip), zone: zone}, port: uint32(port)}, nil
// 		} else {
// 			return nil, &net.AddrError{
// 				Err:  "unsupported address type",
// 				Addr: addr.String(),
// 			}
// 		}
// 	}

// 	switch a := addr.(type) {
// 	case *net.IPAddr:
// 		return ipaddr(a.IP, a.Zone, 0)
// 	case *net.TCPAddr:
// 		return ipaddr(a.IP, a.Zone, a.Port)
// 	case *net.UDPAddr:
// 		return ipaddr(a.IP, a.Zone, a.Port)
// 	case *net.UnixAddr:
// 		return &sockaddrUnix{name: a.Name}, nil
// 	}

// 	return nil, &net.AddrError{
// 		Err:  "unsupported address type",
// 		Addr: addr.String(),
// 	}
// }

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
