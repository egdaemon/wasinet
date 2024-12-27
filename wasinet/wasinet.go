package wasinet

import (
	"net"
	"net/http"
	"net/netip"
	"time"
)

// hijack golang's networks net.DefaultResolver
func Hijack() {
	net.DefaultResolver.Dial = DialContext
	http.DefaultTransport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&Dialer{
			Timeout: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

func netipaddrportToRaw(nap netip.AddrPort) (rawsocketaddr, error) {
	if nap.Addr().Is4() {
		a := sockipaddr[sockip4]{port: uint32(nap.Port()), addr: sockip4{ip: nap.Addr().As4()}}
		return a.sockaddr(), nil
	} else {
		a := sockipaddr[sockip6]{port: uint32(nap.Port()), addr: sockip6{ip: nap.Addr().As16()}}
		return a.sockaddr(), nil
	}
}
