package raopd

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZoneFromIP(t *testing.T) {
	ip := net.IPv4(127, 0, 0, 1)

	x := interfaceNameFromIP(ip)

	assert.Equal(t, "lo", x)
}
