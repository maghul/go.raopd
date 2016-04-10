package raopd

import (
	"fmt"
	"io"
	"os"
	"strconv"
)

// This type is used to register services
type ServiceRegistry struct {
	i *info // actually only crypto stuff
}

type ServiceRef struct {
	raop
}

/*
Create a service registry initialized with the RSA encryption key
used by all services. The keyfile should be in PEM format.
*/
func NewServiceRegistry(keyfile io.Reader) (*ServiceRegistry, error) {
	rf := &ServiceRegistry{}
	var err error
	rf.i, err = makeInfo(keyfile)
	if err != nil {
		return nil, err
	}

	return rf, nil
}

/*
Close all services created in this registry
*/
func (rf *ServiceRegistry) Close() {
}

/*
Creates a new service. port specifies which port the RAOP service should start at
if it zero then an ephemeral port will be used.
*/
func (rf *ServiceRegistry) RegisterService(service Service) (*ServiceRef, error) {
	svc := &ServiceRef{}

	svc.raop.plc = service
	svc.raop.rf = rf
	svc.raop.startRtspProcess()

	svc.raop.br = makeAPBonjourRecord(&svc.raop)
	err := svc.raop.br.Publish()
	if err != nil {
		svc.raop.close()
		return nil, err
	}

	return svc, nil
}

/*
Close the service and remove all published records of the service
*/
func (svc *ServiceRef) Close() {
	fmt.Println("Raop::close")
	svc.br.Unpublish()
}

/*
Returns the port of the RAOP server. This is useful if the service
was created with an ephemeral port, i.e. port==0.
*/
func (svc *ServiceRef) Port() uint16 {
	return svc.port()
}

/*
String returns a brief description of the service. Useful for logging
and debugging.
*/
func (svc *ServiceRef) String() string {
	return svc.raop.String()
}

func (svc *ServiceRef) Command(cmd string) {
	svc.dacp.tx(cmd)
}

func (svc *ServiceRef) Volume(vol string) {
	ivol, err := strconv.ParseFloat(vol, 32)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error converting volume ", vol, " to integer:", err)
	}
	svc.raop.vol.SetDeviceVolume(float32(ivol))
}
