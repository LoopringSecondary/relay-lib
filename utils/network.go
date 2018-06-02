package utils

import (
	"net"
	"strings"
)

func GetLocalIp() string {
	var res = "unknown"
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return res
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				res = ipnet.IP.To4().String()
			}
		}
	}
	return res
}

func GetLocalIpByInterface(name string) string {
	var res = "unknown"
	interfaces, err := net.Interfaces()
	if err != nil {
		return res
	}

	for _, i := range interfaces {
		if name == i.Name {
			if addresses, err := i.Addrs(); err == nil {
				for _, v := range addresses {
					parts := strings.Split(v.String(), ":")
					if len(parts) == 1 { //ipv6 address
						return res
					} else {
						parts := strings.Split(v.String(), "/")
						if len(parts) == 2 {
							return parts[0]
						} else {
							return res
						}
					}
				}
			}
			break
		}
	}
	return res
}