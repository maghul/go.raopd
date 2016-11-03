package raopd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func NoTestZeroconfBrowse(t *testing.T) {
	br := &zeroconfRecord{}
	br.serviceName = "0029B0A72096@PlingPlong"
	br.serviceType = "_knytte._tcp"
	br.serviceDomain = "local" // sdomain
	br.serviceHost = "flurer"
	br.Port = 7777
	publish(br)

	assert.NotNil(t, br)
	time.Sleep(4 * time.Second)

	req, err := zeroconf().resolveService("0029B0A72096@PlingPlong", br.serviceType)
	if err != nil {
		panic(err)
	}
	addr := <-req.result
	zconflog.Debug.Println("Got result: ", addr, addr.addr, addr.txt)
	assert.NotNil(t, addr)

	time.Sleep(14 * time.Second)
	unpublish(br)
	time.Sleep(20 * time.Second)
	zeroconf().zeroconfCleanUp()

}
