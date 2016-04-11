package raopd

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZoneFromIP(t *testing.T) {
	ip := net.IPv4(127, 0, 0, 1)

	x, err := interfaceNameFromIP(ip)
	assert.NoError(t, err)
	assert.Equal(t, "lo", x)
}
