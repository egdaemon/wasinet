package wasip1syscall

import (
	"log"
	"net"
	"syscall"
)

func SocketAddressFormat(family, sotype int) func(sa sockaddr) net.Addr {
	switch int32(family) {
	case AF().INET, AF().INET6:
		switch sotype {
		case syscall.SOCK_STREAM:
			return tcpNetAddr
		case syscall.SOCK_DGRAM:
			return udpNetAddr
		case syscall.SOCK_RAW:
			return ipNetAddr
		}
	case AF().UNIX:
		switch sotype {
		case syscall.SOCK_STREAM:
			return unixNetAddr
		case syscall.SOCK_DGRAM:
			return unixgramNetAddr
		case syscall.SOCK_SEQPACKET:
			return unixpacketNetAddr
		}
	}

	log.Println(family, AF().INET, AF().INET6, AF().UNIX, "|", sotype, syscall.SOCK_STREAM, syscall.SOCK_DGRAM)
	return func(sa sockaddr) net.Addr { return nil }
}

func unixNetAddr(sa sockaddr) net.Addr {
	if sa == nil {
		return &net.UnixAddr{}
	}
	switch proto := sa.(type) {
	case *addressany[addrunix]:
		return &net.UnixAddr{Name: proto.addr.name, Net: "unix"}
	default:
		return nil
	}
}

func unixgramNetAddr(sa sockaddr) net.Addr {
	if sa == nil {
		return &net.UnixAddr{}
	}
	switch proto := sa.(type) {
	case *addressany[addrunix]:
		return &net.UnixAddr{Name: proto.addr.name, Net: "unixgram"}
	default:
		return nil
	}
}

func unixpacketNetAddr(sa sockaddr) net.Addr {
	if sa == nil {
		return &net.UnixAddr{}
	}
	switch proto := sa.(type) {
	case *addressany[addrunix]:
		return &net.UnixAddr{Name: proto.addr.name, Net: "unixpacket"}
	}
	return nil
}

func tcpNetAddr(sa sockaddr) net.Addr {
	if sa == nil {
		return &net.TCPAddr{}
	}

	switch unknown := sa.(type) {
	case *addressany[addrip4]:
		return &net.TCPAddr{IP: unknown.addr.ip[:], Port: int(unknown.addr.port)}
	case *addressany[addrip6]:
		return &net.TCPAddr{IP: unknown.addr.ip[:], Port: int(unknown.addr.port), Zone: ""}
	}
	return nil
}

func udpNetAddr(sa sockaddr) net.Addr {
	if sa == nil {
		return &net.UDPAddr{}
	}

	switch unknown := sa.(type) {
	case *addressany[addrip4]:
		return &net.UDPAddr{IP: unknown.addr.ip[:], Port: int(unknown.addr.port)}
	case *addressany[addrip6]:
		return &net.UDPAddr{IP: unknown.addr.ip[:], Port: int(unknown.addr.port), Zone: ""}
	default:
		return nil
	}
}

func ipNetAddr(sa sockaddr) net.Addr {
	if sa == nil {
		return &net.IPAddr{}
	}

	switch proto := sa.(type) {
	case *addressany[addrip4]:
		return &net.IPAddr{IP: proto.addr.ip[0:]}
	case *addressany[addrip6]:
		return &net.IPAddr{IP: proto.addr.ip[0:], Zone: ""}
	default:
		return nil
	}
}
