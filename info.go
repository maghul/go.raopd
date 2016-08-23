package raopd

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"os"
)

// Contains generic information relevant to all servers/clients
type info struct {
	key *rsa.PrivateKey // Move all code related to this here...
}

var authlog = logger.GetLogger("raopd.auth")

func makeInfo(keyfilename string) (*info, error) {
	i := &info{}
	file, err := os.Open(keyfilename)
	if err != nil {
		return nil, err
	}
	i.key, err = getRSAPrivateKey(file)
	return i, err
}

var ipv4prefix = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff}

func isIPv4(ipaddr net.IP) bool {
	for ii, cb := range ipv4prefix {
		if ipaddr[ii] != cb {
			return false
		}
	}
	return true
}

func ipToBytes(ipaddr net.IP) []byte {
	ipb := []byte(ipaddr)
	if len(ipb) == 4 {
		return ipb
	}
	if isIPv4(ipaddr) {
		return ipaddr[12:16]
	} else {
		return ipaddr
	}
}

func (i *info) decodeBase64(b64 string) ([]byte, error) {
	fmt.Println("base64 string", b64)
	enc := base64.RawStdEncoding
	if b64[len(b64)-1] == '=' {
		enc = base64.StdEncoding
	}

	return enc.DecodeString(b64)
}

func (i *info) rsaKeySign(b64digest string, ipaddr net.IP, hwaddr net.HardwareAddr) (string, error) {
	buffer := bytes.NewBufferString("")

	digest, err := i.decodeBase64(b64digest)
	if err != nil {
		return "", err
	}

	fmt.Println("digest", hex.Dump(digest))
	buffer.Write(digest)
	length := 0
	ipb := ipToBytes(ipaddr)
	fmt.Println("len(ipaddr)=", len(ipaddr))
	if len(ipb) == 4 {
		fmt.Println("ipaddr IPv4", hex.Dump(ipb))
		length = 32
	} else {
		fmt.Println("ipaddr IPv6", hex.Dump(ipb))
		length = 38
	}
	buffer.Write(ipb)
	fmt.Println("hwaddr", hex.Dump(hwaddr))
	buffer.Write(hwaddr)
	buffer.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	bb := buffer.Bytes()[0:length]
	dst, err := rsa.SignPKCS1v15(nil, i.key, 0, bb)
	if err != nil {
		return "", err
	}
	sign := base64.RawStdEncoding.EncodeToString(dst)
	return sign, nil
}

func (i *info) rsaKeyDecrypt(b64encrypted string) ([]byte, error) {
	label := []byte{}

	encrypted, err := i.decodeBase64(b64encrypted)
	if err != nil {
		return nil, err
	}
	decrypted, err := rsa.DecryptOAEP(sha1.New(), nil, i.key, encrypted, label)
	if err != nil {
		return nil, err
	}

	return decrypted, nil
}

func (i *info) rsaKeyParseIv(iv string) ([]byte, error) {
	dec, err := i.decodeBase64(iv)
	if err != nil {
		return nil, err
	}

	return dec, nil
}
