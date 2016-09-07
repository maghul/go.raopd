package raopd

// This uses a Pure Go implementation of Bonjour. It will always
// work but at the cost of having another MDNS responder process
// on the host.

import (
	"fmt"
	"net"
	"strings"

	"github.com/oleksandr/bonjour"
)

type zeroconfPureImplementation struct {
}

func init() {
	registerZeroconfProvider(1000, "Pure Go",
		func() zeroconfImplementation {
			return &zeroconfPureImplementation{}
		})
}

func (bi *zeroconfPureImplementation) fqdn() string {
	return defaultGetMyFQDN()
}

func (bi *zeroconfPureImplementation) zeroconfCleanUp() {
	rs := registeredServers
	registeredServers = nil

	for _, r := range rs {
		zconflog.Debug.Println("Cleanup: Unpublishing! ", r.serviceName, " from service on port=", r.Port)
		s := r.obj.(*bonjour.Server)
		s.Shutdown()
	}
}

func (bi *zeroconfPureImplementation) Unpublish(r *zeroconfRecord) error {
	zconflog.Debug.Println("pure Unpublishing! ", r.serviceName, " from service on port=", r.Port)
	delete(registeredServers, r.serviceName)
	s := r.obj.(*bonjour.Server)
	s.Shutdown()
	return nil
}

func (bi *zeroconfPureImplementation) Publish(r *zeroconfRecord) error {

	zconflog.Debug.Println("Publish: r=", r)
	var err error
	zconflog.Info.Println("zeroconf_pure: Publish r=", r)

	addrs, err := net.LookupIP(r.serviceHost)
	if err != nil {
		// Try appending the host domain suffix and lookup again
		// (required for Linux-based hosts)
		tmpHostName := fmt.Sprintf("%s%s.", r.serviceHost, r.serviceDomain)
		addrs, err = net.LookupIP(tmpHostName)
		if err != nil {
			fmt.Printf("Could not determine host IP addresses for %s", r.serviceHost)
			return fmt.Errorf("Could not determine host IP addresses for %s", r.serviceHost)
		}
	}
	host := fmt.Sprintf("%s.", r.serviceHost)
	ip := fmt.Sprintf("%v", addrs[0])
	r.obj, err = bonjour.RegisterProxy(r.serviceName, r.serviceType, r.serviceDomain, int(r.Port), host, ip, toStringArray(r.text), nil)
	println("---- Error: ", err)
	//r.obj, err = bonjour.Register(r.serviceName, r.serviceType, r.serviceDomain, int(r.Port), toStringArray(r.text), nil)
	if err == nil {
		registeredServers[r.serviceName] = r
	}
	return err
}

// -------------------------- resolve ---------------------------------------------------------------

func (bi *zeroconfPureImplementation) resolveService(srvName, srvType string) (*zeroconfResolveRequest, error) {
	srvName = strings.Replace(srvName, "@", "\\@", -1)
	zconflog.Debug.Println("resolveService: name=", srvName, ", type=", srvType)
	result := make(chan *zeroconfResolveReply, 4)
	req := &zeroconfResolveRequest{zeroconfResolveKey{srvName, srvType}, result, nil}

	resolver, err := bonjour.NewResolver(nil)
	if err != nil {
		return nil, err
	}
	req.resolveObj = resolver
	entriesChannel := make(chan *bonjour.ServiceEntry)
	go func() {
		for {
			zconflog.Debug.Println("resolveService: result...")
			e := <-entriesChannel
			zconflog.Debug.Println("resolveService: result=", e)
			zconflog.Debug.Println("resolveService: result IPv4=", e.AddrIPv4)
			zconflog.Debug.Println("resolveService: result IPv6=", e.AddrIPv6)
			instance := strings.Replace(e.Instance, "\\@", "@", -1)

			if e.AddrIPv4 != nil {
				tcp4 := net.TCPAddr{e.AddrIPv4, e.Port, ""}
				r1 := &zeroconfResolveReply{instance, &tcp4, reworkTxt(e.Text)}
				zconflog.Debug.Println("resolveService: r1=", r1)
				req.result <- r1
			}
			if e.AddrIPv6 != nil {
				tcp6 := net.TCPAddr{e.AddrIPv6, e.Port, ""}
				r2 := &zeroconfResolveReply{instance, &tcp6, reworkTxt(e.Text)}
				zconflog.Debug.Println("resolveService: r2=", r2)
				req.result <- r2
			}
		}
	}()
	resolver.Lookup(srvName, srvType, "local.", entriesChannel)
	return req, nil

}

func (bi *zeroconfPureImplementation) close(req *zeroconfResolveRequest) {
	resolver := req.resolveObj.(*bonjour.Resolver)
	resolver.Exit <- true
}
