package raopd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestZeroconfBrowse(t *testing.T) {
	br := &zeroconfRecord{}
	br.serviceName = "0029B0A72096@PlingPlong"
	br.serviceType = "_knytte._tcp"
	br.serviceDomain = "local" // sdomain
	br.serviceHost = defaultGetMyFQDN()
	br.Port = 7777
	Publish(br)

	time.Sleep(1000000000)
	req, err := zeroconf().resolveService("0029B0A72096@PlingPlong", br.serviceType)
	if err != nil {
		panic(err)
	}
	addr := <-req.result
	zconflog.Debug.Println("Got result: ", addr, addr.addr, addr.txt)
	assert.NotNil(t, addr)
	zeroconf().zeroconfCleanUp()
}
