package raopd

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"os/exec"
	"strings"

	"github.com/guelfey/go.dbus"
)

type bonjourRecord struct {
	serviceName   string
	serviceType   string
	serviceDomain string
	serviceHost   string
	Port          uint16
	text          [][]byte

	obj *dbus.Object
}

func getMyFQDN() string {
	cmd := exec.Command("/bin/hostname", "-f")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Panic(err)
	}
	fqdn := out.String()
	fqdn = fqdn[:len(fqdn)-1] // removing EOL
	if strings.Index(fqdn, ".") < 0 {
		fqdn = fqdn + ".local"
	}
	fmt.Println("FQDN: ", fqdn)
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

	r.serviceName = fmt.Sprintf("%s@%s", hwaddr, raop.plc.ServiceInfo().Name)
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
	fmt.Println("Unpublishing! ", r.serviceName, " from service on port=", r.Port)
	r.obj.Call("org.freedesktop.Avahi.EntryGroup.Free", 0)
	r.obj = nil
}

func (r *bonjourRecord) Publish() error {
	var dconn *dbus.Conn
	var obj *dbus.Object
	var path dbus.ObjectPath
	var err error

	if r.obj != nil {
		return errors.New(fmt.Sprintf("Service '%s' is alread published", r.serviceName))
	}
	dconn, err = dbus.SystemBus()
	if err != nil {
		//		log.Fatal("Fatal error ", err.Error())
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
		return c.Err
	}
	r.obj.Call("org.freedesktop.Avahi.EntryGroup.Commit", 0)
	fmt.Println("Publishing! ", r.serviceName, " as service on port=", r.Port)
	return nil
}
