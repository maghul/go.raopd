// +build !linux

package raopd

import (
	"net"

	"github.com/andrewtj/dnssd"
)

// This calls the Apple(TM) Bonjour(TM) library to resolve
// and publish DNS-SD records
type zeroconfBonjourImplementation struct {
}

func init() {
	registerZeroconfProvider(200, "Bonjour",
		func() zeroconfImplementation {
			return &zeroconfBonjourImplementation{}
		})
}

func (bi *zeroconfBonjourImplementation) fqdn() string {
	return defaultGetMyFQDN()
}

func (bi *zeroconfBonjourImplementation) zeroconfCleanUp() {
}

func (bi *zeroconfBonjourImplementation) Unpublish(r *zeroconfRecord) error {
	return nil
}

func (bi *zeroconfBonjourImplementation) Publish(r *zeroconfRecord) error {
	zconflog.Debug.Println("Publish: r=", r)
	f := func(op *dnssd.RegisterOp, err error, add bool, name, serviceType, domain string) {
		println("name=", name, ", add=", add)
	}

	ro := dnssd.NewRegisterOp(r.serviceName, r.serviceType, int(r.Port), f)
	r.obj = ro
	ro.SetDomain(r.serviceDomain)

	err := ro.Start()
	if err == nil {
		registeredServers[r.serviceName] = r
	}
	return err
}

// -------------------------- resolve ---------------------------------------------------------------

type bonjourResolveData struct {
	r      *dnssd.ResolveOp
	q1, q2 *dnssd.QueryOp
}

func (bi *zeroconfBonjourImplementation) resolveService(srvName, srvType string) (*zeroconfResolveRequest, error) {
	result := make(chan *zeroconfResolveReply, 4)
	resolveData := &bonjourResolveData{}
	req := &zeroconfResolveRequest{zeroconfResolveKey{srvName, srvType}, result, resolveData}

	zconflog.Debug.Println("resolveService: name=", srvName, ", type=", srvType)
	f := func(op *dnssd.ResolveOp, err error, host string, port int, txt map[string]string) {
		if err != nil {
			zconflog.Info.Println("Error Resolving ", srvName, ":", err.Error())
			return
		}
		qf := func(op *dnssd.QueryOp, err error, add bool, interfaceIndex int, fullname string, rrtype, rrclass uint16, rdata []byte, ttl uint32) {
			addr := &net.TCPAddr{IP: net.IP(rdata), Port: port, Zone: ""}
			result <- &zeroconfResolveReply{srvName, addr, txt}
		}
		resolveData.q1, err = dnssd.StartQueryOp(dnssd.InterfaceIndexLocalOnly, host, 1, 1, qf)
		if err != nil {
			zconflog.Info.Println("Error querying IPv4 address for ", host, ":", err.Error())
		}
		resolveData.q2, err = dnssd.StartQueryOp(dnssd.InterfaceIndexLocalOnly, host, 28, 1, qf)
		if err != nil {
			zconflog.Info.Println("Error querying IPv6 address for ", host, ":", err.Error())
		}
	}
	var err error
	resolveData.r, err = dnssd.StartResolveOp(dnssd.InterfaceIndexLocalOnly, srvName, srvType, "local", f)

	return req, err

}

func (bi *zeroconfBonjourImplementation) close(req *zeroconfResolveRequest) {
	resolveData := req.resolveObj.(bonjourResolveData)
	resolveData.r.Stop()
	if resolveData.q1 != nil {
		resolveData.q1.Stop()
	}
	if resolveData.q2 != nil {
		resolveData.q2.Stop()
	}
}
