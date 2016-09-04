package raopd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFQDN(t *testing.T) {
	fqdn := getMyFQDN()
	assert.Equal(t, "durer.local", fqdn)
}

func TestZeroconfBrowse(t *testing.T) {
	br := &zeroconfRecord{}
	br.serviceName = "0009B0A72096@PlingPlong"
	br.serviceType = "_knytte._tcp"
	br.serviceDomain = "local" // sdomain
	br.Port = 7777
	br.Publish()

	req, err := resolveService("0009B0A72096@PlingPlong", "_knytte._tcp")
	if err != nil {
		panic(err)
	}

	addr := <-req.result
	zconflog.Debug.Println("Got result: ", addr, addr.addr)
	assert.NotNil(t, addr)
}
