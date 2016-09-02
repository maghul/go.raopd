package raopd

import (
	"bytes"
	"emh/logger"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

var zconflog = logger.GetLogger("raopd.zeroconf")

type bonjourRecord struct {
	serviceName   string
	serviceType   string
	serviceDomain string
	serviceHost   string
	Port          uint16
	text          [][]byte
	text2         map[string]string

	obj interface{} // Implementation specific object reference
}

var registeredServers = make(map[string]*bonjourRecord)

// Get Fully Qualified Bonjour Domain Name
func getMyFQDN() string {
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

func (b *bonjourRecord) String() string {
	return fmt.Sprintf("BonjourRecord{%s,%s,%s,%s:%d}",
		b.serviceName, b.serviceType, b.serviceDomain, b.serviceHost, b.Port)

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
	txt  []string
}

func toStringArray(d [][]byte) []string {
	r := make([]string, len(d))
	for ii := 0; ii < len(d); ii++ {
		r[ii] = string(d[ii])
	}
	return r
}