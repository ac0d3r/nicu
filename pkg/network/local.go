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

// IPRange calculate the start and end IP addresses according to the subnet mask
// thx: https://github.com/shadow1ng/fscan/blob/main/common/ParseIP.go
func IPRange(c net.IPNet) (net.IP, net.IP) {
	var (
		start, end net.IP
		ipIdx      int
		mask       byte
	)
	start = make(net.IP, len(c.IP))
	end = make(net.IP, len(c.IP))
	copy(start, c.IP)
	copy(end, c.IP)

	for i := 0; i < len(c.Mask); i++ {
		ipIdx = len(end) - 1 - i
		mask = c.Mask[len(c.Mask)-i-1]
		end[ipIdx] = c.IP[ipIdx] | ^mask
		start[ipIdx] = c.IP[ipIdx] & mask
	}

	return start, end
}
