package wasinet

import (
	"net"
	"runtime"
	"strconv"
	"syscall"

	"github.com/egdaemon/wasinet/wasinet/ffi"
	"github.com/egdaemon/wasinet/wasinet/ffierrors"
)

func resolveaddrip(op, network, address string) (res []net.IP, err error) {
	if ip := net.ParseIP(address); ip != nil {
		return []net.IP{ip}, nil
	}

	netip := networkip(network)

	if address == "" && op == oplisten {
		if netip == "ip6" {
			return []net.IP{net.IPv6zero}, nil
		}

		return []net.IP{net.IPv4zero}, nil
	}

	if address == "" {
		if netip == "ip6" {
			return []net.IP{net.IPv6loopback}, nil
		}

		return []net.IP{net.IPv4(127, 0, 0, 1)}, nil
	}

	var (
		bufreslength uint32
	)

	buf := make([]byte, net.IPv6len*8)

	networkptr, networklen := ffi.String(netip)
	addressptr, addresslen := ffi.String(address)
	bufptr, buflen := ffi.Slice(buf)
	bufresptr, _ := ffi.Pointer(&bufreslength)

	errno := sock_getaddrip(
		networkptr,
		networklen,
		addressptr,
		addresslen,
		bufptr,
		buflen,
		bufresptr,
	)
	runtime.KeepAlive(netip)
	runtime.KeepAlive(address)
	runtime.KeepAlive(buf)

	if err = ffierrors.Error(errno); err != nil {
		return nil, err
	}

	for i := 0; i < int(bufreslength); i += net.IPv6len {
		res = append(res, net.IP(buf[i:i+net.IPv6len]))
	}

	return res, nil
}

func resolveport(network, service string) (_port int, err error) {
	var (
		port uint32
	)

	if _port, err = strconv.Atoi(service); err == nil {
		return _port, nil
	}

	networkptr, networklen := ffi.String(network)
	serviceptr, servicelen := ffi.String(service)
	portptr, _ := ffi.Pointer(&port)
	errno := sock_getaddrport(
		networkptr,
		networklen,
		serviceptr,
		servicelen,
		portptr,
	)

	return int(port), syscall.Errno(errno)
}
