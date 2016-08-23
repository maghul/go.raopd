package raopd

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsIPv4(t *testing.T) {

	ipv4 := net.IP([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff, 0x0a, 0x12, 0x34, 0x56})
	assert.True(t, isIPv4(ipv4))

	ipv6 := net.IP([]byte{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff, 0x0a, 0x12, 0x34, 0x56})
	assert.False(t, isIPv4(ipv6))

}
