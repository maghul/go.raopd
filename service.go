package raopd

import (
	"fmt"
	"io"
	"net"
)

// This type is used to register services
type ServiceRegistry struct {
	i *info
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

	si := service.ServiceInfo()

	var r *raop
	r = &svc.raop
	r.dacp = newDacp()

	r.plc = service

	r.audioBuffer = make([]byte, 8192)

	r.hwaddr = si.HardwareAddress

	r.rf = rf

	var err error
	r.l, err = net.Listen("tcp", fmt.Sprintf(":%d", si.Port))
	if err != nil {
		return nil, err
	}

	fmt.Println("Starting RTSP server at ", r.l.Addr())
	s := makeRtspServer(rf.i, r)
	r.rtsp = s
	go s.Serve(r.l)

	r.br = makeAPBonjourRecord(r)
	err = r.br.Publish()
	if err != nil {
		s.Close()
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
Returns a brief description of the service. Useful for logging
and debugging.
*/
func (svc *ServiceRef) String() string {
	return svc.raop.String()
}
