// +build linux

package raopd

import (
	"errors"
	"fmt"
	"os"

	"github.com/guelfey/go.dbus"
)

type zeroconfAvahiImplementation struct {
	dconn  *dbus.Conn
	obj    *dbus.Object
	myfqdn string
}

func init() {
	registerZeroconfProvider(100, "Avahi",
		func() zeroconfImplementation {
			dconn, err := dbus.SystemBus()
			if err != nil {
				zconflog.Debug.Println("dbus.SystemBus error ", err)
				return nil
			}
			obj := dconn.Object("org.freedesktop.Avahi", "/")
			if obj == nil {
				zconflog.Debug.Println("Could not get Avahi DBUS object, probably not available")
				return nil
			}
			hn := obj.Call("org.freedesktop.Avahi.Server.GetHostNameFqdn", 0)
			if hn.Err != nil {
				zconflog.Debug.Println("Error GetHostNameFqdb ", hn.Err.Error())
				return nil
			}
			var fqdn string
			hn.Store(&fqdn)
			zconflog.Debug.Println("Avahi FQDN=", fqdn)
			requestChan = make(chan reqFunc, 5)
			go runResolver(requestChan)
			return &zeroconfAvahiImplementation{dconn, obj, fqdn}
		})
}

func (bi *zeroconfAvahiImplementation) zeroconfCleanUp() {
}

func (bi *zeroconfAvahiImplementation) fqdn() string {
	return bi.myfqdn
}

func (r *zeroconfRecord) dbusObject() *dbus.Object {
	return (r.obj).(*dbus.Object)
}

func (bi *zeroconfAvahiImplementation) Unpublish(r *zeroconfRecord) error {
	zconflog.Debug.Println("avahi Unpublishing! ", r.serviceName, " from service on port=", r.Port)
	r.dbusObject().Call("org.freedesktop.Avahi.EntryGroup.Free", 0)
	r.obj = nil
	return nil
}

func (bi *zeroconfAvahiImplementation) Publish(r *zeroconfRecord) error {
	var path dbus.ObjectPath

	zconflog.Debug.Println("Publish: r=", r)

	if r.obj != nil {
		return errors.New(fmt.Sprintf("Service '%s' is alread published", r.serviceName))
	}

	bi.obj.Call("org.freedesktop.Avahi.Server.EntryGroupNew", 0).Store(&path)
	r.obj = bi.dconn.Object("org.freedesktop.Avahi", path)

	// http://www.dns-sd.org/ServiceTypes.html
	c := r.dbusObject().Call("org.freedesktop.Avahi.EntryGroup.AddService", 0,
		int32(-1),                                            // avahi.IF_UNSPEC
		int32(-1),                                            // avahi.PROTO_UNSPEC
		uint32(0),                                            // flags
		r.serviceName,                                        // sname
		r.serviceType,                                        // stype
		r.serviceDomain,                                      // sdomain
		fmt.Sprintf("%s.%s", r.serviceHost, r.serviceDomain), // shost
		r.Port, // port
		r.text) // text record
	if c.Err != nil {
		zconflog.Debug.Println("org.freedesktop.Avahi.EntryGroup.AddService error ", c.Err.Error())
		return c.Err
	}

	zconflog.Debug.Println("Publishing! ", r.serviceName, " as service on port=", r.Port)
	c = r.dbusObject().Call("org.freedesktop.Avahi.EntryGroup.Commit", 0)
	if c.Err != nil {
		zconflog.Info.Println("org.freedesktop.Avahi.EntryGroup.Commit ", r.serviceName, ", err=", c.Err)
		return c.Err
	}

	return nil
}

// -------------------------- resolve ---------------------------------------------------------------

type reqFunc func(dconn *dbus.Conn, avahi *dbus.Object, requests map[zeroconfResolveKey]*zeroconfResolveRequest)

var requestChan chan reqFunc

func newResolver(dconn *dbus.Conn, avahi *dbus.Object, req *zeroconfResolveRequest) error {
	c := avahi.Call("org.freedesktop.Avahi.Server.ServiceResolverNew", 0,
		int32(-1), // avahi.IF_UNSPEC
		int32(-1), // avahi.PROTO_UNSPEC
		req.srvName,
		req.srvType,
		"local",
		int32(-1), // avahi.PROTO_UNSPEC
		uint32(0))
	if c.Err != nil {
		return c.Err
	}

	var path dbus.ObjectPath
	err := c.Store(&path)
	if err != nil {
		return err
	}
	req.resolveObj = dconn.Object("org.freedesktop.Avahi", path)

	return nil
}

func (req *zeroconfResolveRequest) dbusObject() *dbus.Object {
	return (req.resolveObj).(*dbus.Object)
}

func runResolver(requestChan chan reqFunc) {
	//	browsers := make(map[string]*browser)
	requests := make(map[zeroconfResolveKey]*zeroconfResolveRequest)
	dconn, err := dbus.SystemBus()
	if err != nil {
		zconflog.Info.Println("Error getting DBUS: ", err)
		os.Exit(-1)
	}

	sigchan := make(chan *dbus.Signal, 32)
	dconn.Signal(sigchan)

	avahi := dconn.Object("org.freedesktop.Avahi", "/")

	for {
		select {
		case s := <-sigchan:
			zconflog.Debug.Println("Received signal: ", s)
			switch s.Name {
			case "org.freedesktop.Avahi.ServiceResolver.Found":
				key := zeroconfResolveKey{s.Body[2].(string), s.Body[3].(string)}
				req := requests[key]
				if req != nil {
					// Ok: lets get the IP address and port...
					//					ipp := fmt.Sprintf("%s:%d", s.Body[7], s.Body[8])
					addr, err := toTCPAddr(toString(s.Body[7]), toString(s.Body[8]))
					txt := toStringArray(s.Body[9].([][]byte))
					name := toString(s.Body[5])
					if err == nil {
						req.result <- &zeroconfResolveReply{name, addr, reworkTxt(txt)}
					} else {
						zconflog.Info.Println("Could not resolve address '", addr, "': ", err)
					}
				} else {
					zconflog.Info.Println("not looking for ", key)
				}
			}

		case r := <-requestChan:
			r(dconn, avahi, requests)
		}
	}

}

func getRequestChan() chan reqFunc {
	return requestChan
}

func (bi *zeroconfAvahiImplementation) resolveService(srvName, srvType string) (*zeroconfResolveRequest, error) {
	result := make(chan *zeroconfResolveReply, 4)
	req := &zeroconfResolveRequest{zeroconfResolveKey{srvName, srvType}, result, nil}
	zconflog.Debug.Println("resolveService: name=", srvName, ", type=", srvType)
	getRequestChan() <- func(dconn *dbus.Conn, avahi *dbus.Object, requests map[zeroconfResolveKey]*zeroconfResolveRequest) {
		zconflog.Debug.Println("New Resolve Request: ", req)
		_, exists := requests[req.zeroconfResolveKey]
		if exists {
			zconflog.Info.Println("The request ", req.zeroconfResolveKey, " is already being resolved")
		} else {
			requests[req.zeroconfResolveKey] = req
			newResolver(dconn, avahi, req)
		}
	}
	return req, nil
}

func (bi *zeroconfAvahiImplementation) close(req *zeroconfResolveRequest) {
	getRequestChan() <- func(dconn *dbus.Conn, avahi *dbus.Object, requests map[zeroconfResolveKey]*zeroconfResolveRequest) {
		zconflog.Debug.Println("Delete Resolve Request: ", req)
		_, exists := requests[req.zeroconfResolveKey]
		if exists {
			delete(requests, req.zeroconfResolveKey)
			c := req.dbusObject().Call("org.freedesktop.Avahi.ServiceResolver.Free", 0)
			err := c.Err
			if err != nil {
				zconflog.Info.Println("Error closing ResolveRequest: ", err)
			}
		} else {
			zconflog.Info.Println("The request ", req.zeroconfResolveKey, " doesn't exist!")
		}
	}
}
