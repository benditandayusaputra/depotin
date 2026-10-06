package db

import "net/netip"

func InetFromIP(ip string) *netip.Prefix {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return nil
	}
	prefix := netip.PrefixFrom(addr, addr.BitLen())
	return &prefix
}
