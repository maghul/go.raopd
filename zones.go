package raopd

import (
	"fmt"
	"net"
)

func interfaceNameFromHost(host string) string {
	ip := net.ParseIP(host)
	return interfaceNameFromIP(ip)
}

func interfaceNameFromIP(ip net.IP) string {
	ifs, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	for _, i := range ifs {
		addrs, err := i.Addrs()
		if err != nil {
			panic(err)
		}
		for _, a := range addrs {
			aip := a.(*net.IPNet)
			zone := i.Name
			fmt.Println("i=", i, ", aip=", aip, ", zone=", zone)
			if ip.Equal(aip.IP) {
				return zone
			}
		}
	}
	return ""
}
