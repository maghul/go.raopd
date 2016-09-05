package raopd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFQDN(t *testing.T) {
	fqdn := getMyFQDN()
	assert.Equal(t, "Dali.local", fqdn)
}

func TestZeroconfBrowse(t *testing.T) {
	br := &zeroconfRecord{}
	br.serviceName = "0009B0A72096@PlingPlong"
	br.serviceType = "_knytte._tcp"
	br.serviceDomain = "local" // sdomain
	br.Port = 7777
	zeroconf().Publish(br)

	time.Sleep(2000000000)

	req, err := zeroconf().resolveService("0009B0A72096@PlingPlong", "_knytte._tcp")
	if err != nil {
		panic(err)
	}
	addr := <-req.result
	zconflog.Debug.Println("Got result: ", addr, addr.addr, addr.txt)
	assert.NotNil(t, addr)
	addr = <-req.result
	zconflog.Debug.Println("Got result2: ", addr, addr.addr, addr.txt)
	assert.NotNil(t, addr)
}
