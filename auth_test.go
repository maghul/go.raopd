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

func checkSign(t *testing.T, ip, hw string, challenge string, delta int) string {
	i, err := makeInfo("testdata/airport.key")
	if err != nil {
		panic(err)
	}

	hwaddr, err := net.ParseMAC(hw)
	if err != nil {
		panic(err)
	}

	ipa := net.ParseIP(ip)
	if ipa == nil {
		panic("No IP")
	}

	signed, err := i.rsaKeySign(challenge, ipa, hwaddr)
	assert.NoError(t, err)
	return signed
}

func TestIPv4Authentication(t *testing.T) {
	result := checkSign(t, "10.223.10.33", "33:a1:42:19:01:22", "BXWCsDU/eAxaV+OnvtoHkA==", 0)
	expected := "TBV1mI1+6BSAJ6PjATqI4X8eBeMjV/h7iwOvSbLdKxpDE8pgTP/dcTC9MrB1A3kBOeCrA6F6LiqZnvn8i6gKffimAI/R+3rivr0tk8CbgoeLRC/8W4XY97Gk+CTLHUdZSxrjeSB0i9xOmyKZIUoy/L/n+gc5zDQli0kAWI+fXuwIaaYU2nkUZ5MbpETsa7/Ixk5msUM4t+8C7a0JmfJzYxZmuRlRSp6ohskrO5WhM+Y68kCGvs5prXiq1gH64uX608tfnU15fw8Fd+pzCEI4QgdSuXWooikd2sC8fYhuifchVbslr1wGDUHrVkU2Z4i54zy5l0Wa1d0k2wb2lGm/sQ"
	assert.Equal(t, expected, result)
}

func TestIPv6Authentication(t *testing.T) {
	result := checkSign(t, "fe80::461e:a1ff:fece:f4a9", "33:a1:42:19:01:22", "BXWCsDU/eAxaV+OnvtoHkA==", 0)
	expected := "axX9wtcsVpCvzokq7tFQWq0dbh6Km40LWcUeJyVjLIuE+wWUJTa7W85w6vjqz2fhctfIFeZdyw34GiwwQaD7fn46c/4VSy4FFLjVFyqWBsEIqnJte8e6v8P8WSHlVE8sveST+ZlUlcda3pqTQ3G1kptrjYz3hvMO55uOh8wY+7T+4vR1zQs9Pxmg96IrZdPV2JzB8o3ardULFBJjvdJMgtq77O6vKkZ8rKAl9svN1Zsj0HognoW2v2FSvZcbJdP6yvvowY6YEEL5wiIVKZhfHUzN+kJXl5jpbbnYnlCIBXd5qzaU/Pf2UKIzlrSOUJ5Uge5PXWOwWUuKadY4c+ykOA"
	assert.Equal(t, expected, result)
}

func TestIPv4Authentication2(t *testing.T) {
	expected := "3nPmjMOHIbFA3Hmc1MPqdW7gAuOL/eY4yd9ARkXCOWxfFHZlo3yjotTMYYmsJd5l+NSlWx9RMTegpM7FYObI0uio46hbfyXRC0Nctu0xiobIxG9Ji8+0L1DOiqSjRjHkAvQzPAvxxwMaTfL6Ug3RRtsG8RPWabTBBwKW6/tAOslKl2XxYPnjK2kYnoVbkcmc3New9kjS0WhEweyX12xonfebFaI8ry4i7NPxcATFc3RjbzlfuvPVdfdMTzhbJ9Nxfq6viQQaafabD3S0anJbuk/cwsxV+3UtDd/9LQXNQFUjcPaJ706LPWFsEQdlvEl+qOUmchYAjkDRAcej0hOJXQ"
	result := checkSign(t, "10.223.10.146", "48:5D:60:7C:EE:22", "6cBX05ZjFasyvTQ7A0m89A", 6)
	assert.Equal(t, expected, result)
}

func TestIPv6Authentication2(t *testing.T) {
	expected := "0Ilw7B5Lmnj57jlXCTJODDS1ecu+vikKHoaBP4ljzef8n1NHrvT2SbJ17wTv7KcQL2a1stxD43c63htWceXUj/7cxiOV6uKWB5ATKaWdOs7NVJblR4KUo2XMxnUUpIesqw+Jrj1VDNj0NVTjtK6aqP4z1CrZkbD9ewWJeVEksmTvz9GQ61HyKli58xQpNtjTDp2JIs+j9cxVEFscej76zLcDtYV1dIB4V4nXXWXXrnskpEG8KTzMKvMvf2ppNn6YxfuP+CycQMurToEVVjE/gMQc086ELEBuBHxXOAmMhxDNHgSsaOfat6YBOCPWP7W9GCLH8qsMBmu9a4zj1hQ6Vw"
	result := checkSign(t, "fe80::0a11:96ff:fe1c:a8bc", "48:5D:60:7C:EE:22", "NyPyw2gyxYEnxpXvRJaNXA==", 6)
	assert.Equal(t, expected, result)
}

func TestAuthIPv4Shairplay(t *testing.T) {
	expected := "wjTaHnZondBktQ7v9PjwoOLtA2kiS9IHl7zouRkFcsREejtAdR/FgcoCWmHSTCysnNmDQhWUAhkcNHloRA6K+Tw+0J2Xv3wg8nfPgiKxcHSqQpVwzKACsh8/7ssBO0hY6E60RbIO2N6pJJqgTj9Xiyd6UMrLMPNFMjgpJt/sDCuUwS8c62yHqu2X6Fhe7vJNGxYAMwCkZIFFwc8U0H3OK5QiWPs4yap9qwL6dEVjmf8BCkZZtUoPDGHsDC9MynJDFAotT7eHUYjbSJkt7boQvr0dIG7zTR4X8vNk9tkWbLdw8GjI966wTiCtf1Xde1nCk92pk2LurHl1JXcKDpKDgg"
	result := checkSign(t, "10.223.10.110", "48:5D:60:7C:EE:22", "az5FIXrxftZarFq0tav2/A==", 6)
	assert.Equal(t, expected, result)
}
