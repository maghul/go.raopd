package raopd

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestZeroconfBrowse(t *testing.T) {
	br := &bonjourRecord{}
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
