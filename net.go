package raopd

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

var netlog = getLogger("raopd.net", "Low level networking")

func getPortsFromTransport(transport string) (control int, timing int, err error) {
	ts := strings.Split(transport, ";")
	for _, tv := range ts {
		var vp = (*int)(nil)
		switch {
		case strings.Index(tv, "control_port=") == 0:
			vp = &control
		case strings.Index(tv, "timing_port=") == 0:
			vp = &timing
		}

		if vp != nil {
			kv := strings.Split(tv, "=")
			var v int64
			v, err = strconv.ParseInt(kv[1], 10, 0)
			if err != nil {
				return
			}
			*vp = int(v)
		}
	}
	return
}

func cToIP(host string) (net.IP, error) {
	if strings.Index(host, "IN IP6 ") != 0 && strings.Index(host, "IN IP4 ") != 0 {
		return nil, errors.New(fmt.Sprintf("Unknown C record '%s'", host))
	}
	a := net.ParseIP(host[7:])
	netlog.Debug.Println("HOST IS", a)
	if a == nil {
		return nil, errors.New(fmt.Sprintf("Unparsable IP address '%s'", host[7:]))
	}
	return a, nil
}
