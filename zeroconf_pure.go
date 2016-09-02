// +build windows

package raopd

// This uses a Pure Go implementation of Bonjour. It will always
// work but at the cost of having another MDNS responder process
// on the host.

import (
	"net"

	"github.com/oleksandr/bonjour"
)

func init() {

}

func zeroconfCleanUp() {
	rs := registeredServers
	registeredServers = nil

	for _, r := range rs {
		zconflog.Debug.Println("Cleanup: Unpublishing! ", r.serviceName, " from service on port=", r.Port)
		s := r.obj.(*bonjour.Server)
		s.Shutdown()
	}
}

func (r *bonjourRecord) Unpublish() {
	zconflog.Debug.Println("Unpublishing! ", r.serviceName, " from service on port=", r.Port)
	delete(registeredServers, r.serviceName)
	s := r.obj.(*bonjour.Server)
	s.Shutdown()
}

func (r *bonjourRecord) Publish() error {

	zconflog.Debug.Println("Publish: r=", r)
	// TODO: This will send the hostname without the .local suffix which AirPlay
	//       devices doesn't seem to like.
	var err error
	r.obj, err = bonjour.Register(r.serviceName, r.serviceType, r.serviceDomain, int(r.Port), toStringArray(r.text), nil)
	if err == nil {
		registeredServers[r.serviceName] = r
	}
	return err
}

// -------------------------- resolve ---------------------------------------------------------------

func resolveService(srvName, srvType string) (*zeroconfResolveRequest, error) {
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
			e := <-entriesChannel
			zconflog.Debug.Println("resolveService: result=", e)
			tcp4 := net.TCPAddr{e.AddrIPv4, e.Port, ""}
			r1 := &zeroconfResolveReply{e.Instance, &tcp4, e.Text}
			req.result <- r1
			if e.AddrIPv4 == nil {
				tcp6 := net.TCPAddr{e.AddrIPv6, e.Port, ""}
				r2 := &zeroconfResolveReply{e.Instance, &tcp6, e.Text}
				req.result <- r2
			}
		}
	}()
	resolver.Lookup(srvName, srvType, "local", entriesChannel)
	return req, nil

}

func (req *zeroconfResolveRequest) close() {
	resolver := req.resolveObj.(*bonjour.Resolver)
	resolver.Exit <- true
}
