package network

import (
	"net"
	"strings"
)

func GetLocalIPV4Net() ([]net.IPNet, error) {
	inters, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	ipnets := make([]net.IPNet, 0)
	for _, inter := range inters {
		if inter.Flags&net.FlagUp == 0 || strings.HasPrefix(inter.Name, "lo") {
			continue
		}
		addrs, err := inter.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
				ipnets = append(ipnets, *ipnet)
			}
		}
	}
	return ipnets, err
}
