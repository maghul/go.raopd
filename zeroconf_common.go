package raopd

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
)

var zconflog = getLogger("raopd.zeroconf", "Zero Configuration")

type zeroconfRecord struct {
	serviceName   string
	serviceType   string
	serviceDomain string
	serviceHost   string
	Port          uint16
	text          [][]byte
	text2         map[string]string

	obj interface{} // Implementation specific object reference
}

type zeroconfImplementation interface {
	Publish(r *zeroconfRecord) error
	Unpublish(r *zeroconfRecord) error
	resolveService(srvName, srvType string) (*zeroconfResolveRequest, error)
	close(*zeroconfResolveRequest)
	zeroconfCleanUp()
}

var registeredServers = make(map[string]*zeroconfRecord)

// Get Fully Qualified Bonjour Domain Name
func defaultGetMyFQDN() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return fmt.Sprintf("%s.local", hostname)
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

func (b *zeroconfRecord) String() string {
	return fmt.Sprintf("BonjourRecord{%s,%s,%s,%s:%d}",
		b.serviceName, b.serviceType, b.serviceDomain, b.serviceHost, b.Port)

}

func (br *zeroconfRecord) appendText(v ...string) {
	vl := len(v)
	for ii := 0; ii < vl; ii++ {
		br.text = append(br.text, bytes.NewBufferString(v[ii]).Bytes())
	}
}

func (zr *zeroconfRecord) txtAsStringArray() []string {
	txt := make([]string, len(zr.text))
	ii := 0
	for _, keyValue := range zr.text {
		txt[ii] = string(keyValue)
		ii++
	}
	return txt
}

func hardwareAddressToServicePrefix(hwaddr net.HardwareAddr) string {
	s := hwaddr.String()
	s = strings.Replace(s, ":", "", -1)
	s = strings.ToUpper(s)
	return s

}

func makeAPBonjourRecord(raop *raop) *zeroconfRecord {
	r := &zeroconfRecord{}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}
	hwaddr := hardwareAddressToServicePrefix(raop.hwaddr)
	port := raop.port()

	r.serviceName = fmt.Sprintf("%s@%s", hwaddr, raop.sink.Info().Name)
	r.serviceType = "_raop._tcp"
	r.serviceDomain = "local" // sdomain
	r.serviceHost = hostname  // shost
	r.Port = port

	version := "0.1" // Get from RAOP or caller.
	r.appendText(
		"txtvers=1",
		"ch=2",     // 2 channels
		"cn=0,1",   // PCM,ALAC
		"et=0,1",   // Encryption, none,RSA
		"sv=false", //
		"da=true",  //
		"am=Squareplay",
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

// -------------------------- resolve ---------------------------------------------------------------

type zeroconfResolveKey struct {
	srvName string
	srvType string
}

type zeroconfResolveRequest struct {
	zeroconfResolveKey
	result     chan *zeroconfResolveReply
	resolveObj interface{}
}

type zeroconfResolveReply struct {
	name string
	addr *net.TCPAddr
	txt  map[string]string
}

func toStringArray(d [][]byte) []string {
	r := make([]string, len(d))
	for ii := 0; ii < len(d); ii++ {
		r[ii] = string(d[ii])
	}
	return r
}

//  ------------------------- Providers ---------------------------------------------

type zeroconfProvider struct {
	priority int
	name     string
	factory  func() zeroconfImplementation
}

type zeroconfProviders []*zeroconfProvider

func (a zeroconfProviders) Len() int           { return len(a) }
func (a zeroconfProviders) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a zeroconfProviders) Less(i, j int) bool { return a[i].priority < a[j].priority }

var registeredProviders = zeroconfProviders{}

func registerZeroconfProvider(prio int, name string, factory func() zeroconfImplementation) {
	zcp := &zeroconfProvider{prio, name, factory}
	registeredProviders = append(registeredProviders, zcp)
}

var _zeroconf zeroconfImplementation

func zeroconf() zeroconfImplementation {
	if _zeroconf != nil {
		return _zeroconf
	}
	zconflog.Info.Println("Could not find any working ZeroConf libraries")
	os.Exit(-1)
	return nil
}

func reworkTxt([]string) map[string]string {
	return nil
}

func unpublish(zr *zeroconfRecord) error {
	return zeroconf().Unpublish(zr)
}

func publish(zr *zeroconfRecord) error {
	zconflog.Debug.Println("Trying to publish ", zr)

	if _zeroconf == nil {
		sort.Sort(registeredProviders)
		for _, p := range registeredProviders {
			zconflog.Debug.Println("Trying to start ", p.name, " Zeroconf provider")
			_zeroconf = p.factory()
			if _zeroconf != nil {
				zconflog.Info.Println("Started ", p.name, " Zeroconf provider")
				err := _zeroconf.Publish(zr)
				if err == nil {
					zconflog.Info.Println("Published ", zr, " with ", p.name, " Zeroconf provider")
					return nil
				}
			}
		}
	} else {
		err := _zeroconf.Publish(zr)
		if err != nil {
			return err
		}
		zconflog.Info.Println("Published ", zr, " established Zeroconf provider")
		return nil
	}
	return errors.New("Could not find any working ZeroConf libraries")
}
