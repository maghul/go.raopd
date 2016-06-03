package raopd

import (
	"bytes"
	"emh/logger"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/guelfey/go.dbus"
)

var zconflog = logger.GetLogger("raopd.zeroconf")

type bonjourRecord struct {
	serviceName   string
	serviceType   string
	serviceDomain string
	serviceHost   string
	Port          uint16
	text          [][]byte

	obj *dbus.Object
}

func init() {
	requestChan = make(chan reqFunc, 5)
	go runResolver(requestChan)
}

func (b *bonjourRecord) String() string {
	return fmt.Sprintf("BonjourRecord{%s,%s,%s,%s:%d}",
		b.serviceName, b.serviceType, b.serviceDomain, b.serviceHost, b.Port)

}

func getMyFQDN() string {
	cmd := exec.Command("/bin/hostname", "-f")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		zconflog.Info.Println(err)
		panic(err)
	}
	fqdn := out.String()
	fqdn = fqdn[:len(fqdn)-1] // removing EOL
	if strings.Index(fqdn, ".") < 0 {
		fqdn = fqdn + ".local"
	}
	zconflog.Debug.Println("FQDN: ", fqdn)
	return fqdn
}

func (br *bonjourRecord) appendText(v ...string) {
	vl := len(v)
	for ii := 0; ii < vl; ii++ {
		br.text = append(br.text, bytes.NewBufferString(v[ii]).Bytes())
	}
}

func hardwareAddressToServicePrefix(hwaddr net.HardwareAddr) string {
	s := hwaddr.String()
	s = strings.Replace(s, ":", "", -1)
	s = strings.ToUpper(s)
	return s

}

func makeAPBonjourRecord(raop *raop) *bonjourRecord {
	r := &bonjourRecord{}

	fqdn := getMyFQDN()
	hwaddr := hardwareAddressToServicePrefix(raop.hwaddr)
	port := raop.port()

	r.serviceName = fmt.Sprintf("%s@%s", hwaddr, raop.sink.Info().Name)
	r.serviceType = "_raop._tcp"
	r.serviceDomain = "local" // sdomain
	r.serviceHost = fqdn      // shost
	r.Port = port

	version := "0.1" // Get from RAOP or caller.
	r.appendText(
		"txtvers=1",
		"ch=2",     // 2 channels
		"cn=0,1",   // PCM,ALAC
		"et=0,1",   // Encryption, none,RSA
		"sv=false", //
		"da=true",  //
		"am=Pairlay",
		"sr=44100",    // Sample Rate
		"ss=16",       // Sample Size
		"pw=false",    // No password
		"vn=3",        //
		"tp=UDP",      // Transports: UDP
		"md=0,1,2",    // Metadata: text, artwork, progress
		"vs="+version, // Version
		"sm=false",    //
		"ek=1")        //

	return r
}

func (r *bonjourRecord) Unpublish() {
	zconflog.Debug.Println("Unpublishing! ", r.serviceName, " from service on port=", r.Port)
	r.obj.Call("org.freedesktop.Avahi.EntryGroup.Free", 0)
	r.obj = nil
}

func (r *bonjourRecord) Publish() error {
	var dconn *dbus.Conn
	var obj *dbus.Object
	var path dbus.ObjectPath
	var err error

	zconflog.Debug.Println("Publish: r=", r)

	if r.obj != nil {
		return errors.New(fmt.Sprintf("Service '%s' is alread published", r.serviceName))
	}
	dconn, err = dbus.SystemBus()
	if err != nil {
		zconflog.Debug.Println("dbus.SystemBus error ", err)
		return err
	}

	obj = dconn.Object("org.freedesktop.Avahi", "/")
	obj.Call("org.freedesktop.Avahi.Server.EntryGroupNew", 0).Store(&path)

	r.obj = dconn.Object("org.freedesktop.Avahi", path)

	// http://www.dns-sd.org/ServiceTypes.html
	c := r.obj.Call("org.freedesktop.Avahi.EntryGroup.AddService", 0,
		int32(-1),       // avahi.IF_UNSPEC
		int32(-1),       // avahi.PROTO_UNSPEC
		uint32(0),       // flags
		r.serviceName,   // sname
		r.serviceType,   // stype
		r.serviceDomain, // sdomain
		r.serviceHost,   // shost
		r.Port,          // port
		r.text)          // text record
	if c.Err != nil {
		zconflog.Debug.Println("org.freedesktop.Avahi.EntryGroup.AddService error ", c.Err.Error())
		return c.Err
	}

	zconflog.Debug.Println("Publishing! ", r.serviceName, " as service on port=", r.Port)
	c = r.obj.Call("org.freedesktop.Avahi.EntryGroup.Commit", 0)
	if c.Err != nil {
		zconflog.Info.Println("org.freedesktop.Avahi.EntryGroup.Commit ", r.serviceName, ", err=", c.Err)
		return c.Err
	}

	return nil
}

// -------------------------- resolve ---------------------------------------------------------------

type zeroconfResolveKey struct {
	srvName string
	srvType string
}

type zeroconfResolveRequest struct {
	zeroconfResolveKey
	result     chan *zeroconfResolveReply
	resolveObj *dbus.Object
}

type zeroconfResolveReply struct {
	name string
	addr *net.TCPAddr
	txt  []string
}

//var requestChan chan *zeroconfResolveRequest
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

func toStringArray(d [][]byte) []string {
	r := make([]string, len(d))
	for ii := 0; ii < len(d); ii++ {
		r[ii] = string(d[ii])
	}
	return r
}

func toTCPAddr(addr, port string) (*net.TCPAddr, error) {
	ip := net.ParseIP(addr)
	p, err := strconv.ParseInt(port, 10, 0)
	if err != nil {
		return nil, err
	}
	a := &net.TCPAddr{IP: ip, Port: int(p), Zone: ""}
	return a, nil
}

func toString(i interface{}) string {
	s := fmt.Sprintf("%v", i)
	return s
}

func runResolver(requestChan chan reqFunc) {
	//	browsers := make(map[string]*browser)
	requests := make(map[zeroconfResolveKey]*zeroconfResolveRequest)
	dconn, err := dbus.SystemBus()
	if err != nil {
		zconflog.Info().Println(os.Stderr, "Error getting DBUS: ", err)
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
						req.result <- &zeroconfResolveReply{name, addr, txt}
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

func resolveService(srvName, srvType string) (*zeroconfResolveRequest, error) {
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

func (req *zeroconfResolveRequest) close() {
	getRequestChan() <- func(dconn *dbus.Conn, avahi *dbus.Object, requests map[zeroconfResolveKey]*zeroconfResolveRequest) {
		zconflog.Debug.Println("Delete Resolve Request: ", req)
		_, exists := requests[req.zeroconfResolveKey]
		if exists {
			delete(requests, req.zeroconfResolveKey)
			c := req.resolveObj.Call("org.freedesktop.Avahi.ServiceResolver.Free", 0)
			err := c.Err
			if err != nil {
				zconflog.Info.Println(os.Stderr, "Error closing ResolveRequest: ", err)
			}
		} else {
			zconflog.Info.Println("The request ", req.zeroconfResolveKey, " doesn't exist!")
		}
	}
}
