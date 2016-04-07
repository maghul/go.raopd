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

func checkSign(ipaddr net.IP) string {
	i, err := makeInfo(getKeyfile())
	if err != nil {
		panic(err)
	}

	challenge := "BXWCsDU/eAxaV+OnvtoHkA=="
	hwaddr := net.HardwareAddr([]byte{0x33, 0xa1, 0x42, 0x19, 0x01, 0x22})

	return i.rsaKeySign(challenge, ipaddr, hwaddr)
}

func TestIPv4Authentication(t *testing.T) {
	ipaddr := net.ParseIP("10.223.10.33")
	result := checkSign(ipaddr)
	expected := "TBV1mI1+6BSAJ6PjATqI4X8eBeMjV/h7iwOvSbLdKxpDE8pgTP/dcTC9MrB1A3kBOeCrA6F6LiqZnvn8i6gKffimAI/R+3rivr0tk8CbgoeLRC/8W4XY97Gk+CTLHUdZSxrjeSB0i9xOmyKZIUoy/L/n+gc5zDQli0kAWI+fXuwIaaYU2nkUZ5MbpETsa7/Ixk5msUM4t+8C7a0JmfJzYxZmuRlRSp6ohskrO5WhM+Y68kCGvs5prXiq1gH64uX608tfnU15fw8Fd+pzCEI4QgdSuXWooikd2sC8fYhuifchVbslr1wGDUHrVkU2Z4i54zy5l0Wa1d0k2wb2lGm/sQ"
	assert.Equal(t, expected, result)
}

func TestIPv6Authentication(t *testing.T) {
	ipaddr := net.ParseIP("fe80::461e:a1ff:fece:f4a9")
	result := checkSign(ipaddr)
	expected := "axX9wtcsVpCvzokq7tFQWq0dbh6Km40LWcUeJyVjLIuE+wWUJTa7W85w6vjqz2fhctfIFeZdyw34GiwwQaD7fn46c/4VSy4FFLjVFyqWBsEIqnJte8e6v8P8WSHlVE8sveST+ZlUlcda3pqTQ3G1kptrjYz3hvMO55uOh8wY+7T+4vR1zQs9Pxmg96IrZdPV2JzB8o3ardULFBJjvdJMgtq77O6vKkZ8rKAl9svN1Zsj0HognoW2v2FSvZcbJdP6yvvowY6YEEL5wiIVKZhfHUzN+kJXl5jpbbnYnlCIBXd5qzaU/Pf2UKIzlrSOUJ5Uge5PXWOwWUuKadY4c+ykOA"
	assert.Equal(t, expected, result)
}
