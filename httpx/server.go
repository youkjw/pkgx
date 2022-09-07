package httpx

import (
	"context"
	"net"
)

type Server interface {
	Start(context.Context) error
	Stop(context.Context) error
}

//type Instance interface {
//	ID() string
//	Name() string
//	Version() string
//	Endpoint() []string
//}

func ExtractEndpoint(address string) (string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", err
	}

	ipNet := net.ParseIP(address)
	if len(host) > 0 && !ipNet.IsLoopback() && host != "0.0.0.0" && host != "::" {
		return net.JoinHostPort(host, port), nil
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, iaddr := range addrs {
		if ipNet, ok := iaddr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil && isValidIP(ipNet.IP.String()) {
				return net.JoinHostPort(ipNet.IP.String(), port), nil
			}
		}
	}

	return "", nil
}

func isValidIP(addr string) bool {
	ip := net.ParseIP(addr)
	return ip.IsGlobalUnicast() && !ip.IsInterfaceLocalMulticast()
}
