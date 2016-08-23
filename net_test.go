package raopd

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNetGetPortsFromTransport(t *testing.T) {
	transport := "RTP/AVP/UDP;unicast;interleaved=0-1;mode=record;control_port=6001;timing_port=6002"

	cp, tp, err := getPortsFromTransport(transport)

	assert.Nil(t, err)
	assert.Equal(t, 6001, cp)
	assert.Equal(t, 6002, tp)

}

func TestNetCToIP(t *testing.T) {
	remote, err := cToIP("IN IP6 fe80::1c14:b58a:3cb8:868b")
	assert.Nil(t, err)
	expected := net.ParseIP("fe80::1c14:b58a:3cb8:868b")
	assert.Equal(t, expected, remote)

	remote, err = cToIP("IN IP4 194.128.55.78")
	assert.Nil(t, err)
	expected = net.ParseIP("194.128.55.78")
	assert.Equal(t, expected, remote)

	remote, err = cToIP("IN IP8 194.128.55.78")
	assert.NotNil(t, err)

	remote, err = cToIP("IN IP4 194.128,55.78")
	assert.NotNil(t, err)
}
