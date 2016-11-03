package raopd

// This uses a Pure Go implementation of Bonjour. It will always
// work but at the cost of having another MDNS responder process
// on the host.

import (
	"context"
	"net"
	"time"

	"github.com/maghul/go.dnssd"
	"github.com/miekg/dns"
)

type zeroconfPureImplementation struct {
}

type zeroconfPureService struct {
	cancel context.CancelFunc
}

func init() {
	registerZeroconfProvider(1000, "Pure Go",
		func() zeroconfImplementation {
			return &zeroconfPureImplementation{}
		})
	/*
		raoplogg := func(data ...interface{}) {
			s := fmt.Sprint(data)
			if strings.Contains(s, "raop") {
				zconflog.Debug.Println("QUERIES", data)
			}
		}
		slf.GetLogger("q").SetOutputLogger(raoplogg)
		logg := func(data ...interface{}) {
			zconflog.Debug.Println(data)
		}
		slf.GetLogger("dnssd").SetOutputLogger(logg)
	*/
}

func (bi *zeroconfPureImplementation) fqdn() string {
	return defaultGetMyFQDN()
}

func (bi *zeroconfPureImplementation) zeroconfCleanUp() {
	rs := registeredServers
	registeredServers = nil

	for _, r := range rs {
		zconflog.Debug.Println("Cleanup: Unpublishing! ", r.serviceName, " from service on port=", r.Port)
		c := r.obj.(*zeroconfPureService)
		c.cancel()
	}
	// TODO: We should wait for cleanup to complete, not timeout...
	time.Sleep(3 * time.Second)
}

func (bi *zeroconfPureImplementation) Unpublish(r *zeroconfRecord) error {
	zconflog.Debug.Println("pure Unpublishing! ", r.serviceName, " from service on port=", r.Port)
	delete(registeredServers, r.serviceName)
	c := r.obj.(*zeroconfPureService)
	c.cancel()
	return nil
}

func (bi *zeroconfPureImplementation) Publish(r *zeroconfRecord) error {

	zconflog.Debug.Println("pure Publish: r=", r)
	var err error
	ctx, cancel := context.WithCancel(context.Background())
	r.obj = &zeroconfPureService{cancel}
	registeredServers[r.serviceName] = r
	dnssd.Register(ctx, 0, 0, r.serviceName, r.serviceType, r.serviceDomain, r.serviceHost, r.Port, r.txtAsStringArray(),
		func(flags int, serviceName, regType, domain string) {
			zconflog.Info.Println("zeroconf_pure: Completed publish r=", r)
		}, func(err error) {
			zconflog.Info.Println("zeroconf_pure: Error publishing r=", r, ", error=", err)
		})
	return err
}

// -------------------------- resolve ---------------------------------------------------------------

func (bi *zeroconfPureImplementation) resolveService(srvName, srvType string) (*zeroconfResolveRequest, error) {
	//	srvName = strings.Replace(srvName, "@", "\\@", -1)
	zconflog.Debug.Println("resolveService: name=", srvName, ", type=", srvType)
	result := make(chan *zeroconfResolveReply, 4)
	req := &zeroconfResolveRequest{zeroconfResolveKey{srvName, srvType}, result, nil}

	errc := func(err error) {
	}

	ctx, cancel := context.WithCancel(context.Background())
	dnssd.Resolve(ctx, 0, 0, srvName, srvType, "",
		func(flags dnssd.Flags, ifIndex int, fullName, hostName string, port uint16, txt []string) {
			zconflog.Debug.Println("zeroconf resolved: ifIndex=", ifIndex, ",serviceName=", fullName, ", hostName=", hostName, ":", port)
			cancel()

			ctx, cancel = context.WithCancel(context.Background())
			dnssd.Query(ctx, 0, ifIndex, &dns.Question{Name: hostName, Qtype: dns.TypeA, Qclass: dns.ClassINET},
				func(flags dnssd.Flags, ifIndex int, rr dns.RR) {
					a := rr.(*dns.A)
					tcpAddr := net.TCPAddr{IP: a.A, Port: int(port), Zone: ""}
					zconflog.Debug.Println("zeroconf pure question: ifIndex=", ifIndex, ",serviceName=", fullName, ", hostName=", hostName, ":", port, ", tcpAddr=", tcpAddr)
					req.result <- &zeroconfResolveReply{hostName, &tcpAddr, reworkTxt(txt)}
					cancel()
				}, errc)

			ctx, cancel = context.WithCancel(context.Background())
			dnssd.Query(ctx, 0, ifIndex, &dns.Question{Name: hostName, Qtype: dns.TypeAAAA, Qclass: dns.ClassINET},
				func(flags dnssd.Flags, ifIndex int, rr dns.RR) {
					aaaa := rr.(*dns.AAAA)
					tcpAddr := net.TCPAddr{IP: aaaa.AAAA, Port: int(port), Zone: ""}
					zconflog.Debug.Println("zeroconf pure question: ifIndex=", ifIndex, ",serviceName=", fullName, ", hostName=", hostName, ":", port, ", tcpAddr=", tcpAddr)
					req.result <- &zeroconfResolveReply{hostName, &tcpAddr, reworkTxt(txt)}
					cancel()
				}, errc)

		}, errc)
	return req, nil

}

func (bi *zeroconfPureImplementation) close(req *zeroconfResolveRequest) {
}
