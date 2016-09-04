// +build linux

package raopd

import (
	"github.com/andrewtj/dnssd"
)

type zeroconfBonjourImplementation struct {
}

func init() {
	zeroconf = &zeroconfBonjourImplementation{}
}

// This calls the Apple(TM) Bonjour(TM) library to resolve
// and publish DNS-SD records
func (bi *zeroconfBonjourImplementation) zeroconfCleanUp() {
}

func (bi *zeroconfBonjourImplementation) Unpublish(r *zeroconfRecord) {
}

func (bi *zeroconfBonjourImplementation) Publish(r *zeroconfRecord) error {

	zconflog.Debug.Println("Publish: r=", r)
	f := func(op *dnssd.RegisterOp, err error, add bool, name, serviceType, domain string) {
		println("name=", name)
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

func (bi *zeroconfBonjourImplementation) resolveService(srvName, srvType string) (*zeroconfResolveRequest, error) {
	return nil, nil

}

func (bi *zeroconfBonjourImplementation) close(req *zeroconfResolveRequest) {
}
