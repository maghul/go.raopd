package raopd

import (
	"errors"
	"fmt"
	"net"
)

func interfaceNameFromHost(host string) (string, error) {
	ip := net.ParseIP(host)
	if ip == nil {
		return "", errors.New(fmt.Sprint("Could not parse IP for host=", host))
	}
	return interfaceNameFromIP(ip)
}

func interfaceNameFromIP(ip net.IP) (string, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, i := range ifs {
		addrs, err := i.Addrs()
		if err != nil {
			return "", err
		}
		for _, a := range addrs {
			aip := a.(*net.IPNet)
			zone := i.Name
			netlog.Debug().Println("i=", i, ", aip=", aip, ", zone=", zone)
			if ip.Equal(aip.IP) {
				return zone, nil
			}
		}
	}
	return "", errors.New(fmt.Sprint("Could not find any interface for address=", ip))
}
