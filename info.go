package raopd

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net"
)

// Contains generic information relevant to all servers/clients
type info struct {
	key *rsa.PrivateKey // Move all code related to this here...
}

func makeInfo(keyfile io.Reader) (*info, error) {
	var err error
	i := &info{}
	i.key, err = getRSAPrivateKey(keyfile)
	if err != nil {
		return nil, err
	}
	return i, nil
}

func (i *info) rsaKeySign(b64digest string, ipaddr net.IP, hwaddr net.HardwareAddr) (string, error) {
	buffer := bytes.NewBufferString("")

	digest, err := base64.StdEncoding.DecodeString(b64digest)
	if err != nil {
		return "", err
	}

	fmt.Println("digest", hex.Dump(digest))
	buffer.Write(digest)
	// TODO: An IPv4 address wont work
	fmt.Println("ipaddr", hex.Dump(ipaddr))
	buffer.Write(ipaddr)
	fmt.Println("hwaddr", hex.Dump(hwaddr))
	buffer.Write(hwaddr)
	bb := buffer.Bytes()[0:buffer.Len()]
	dst, err := rsa.SignPKCS1v15(nil, i.key, 0, bb)
	if err != nil {
		return "", err
	}
	sign := base64.RawStdEncoding.EncodeToString(dst)
	return sign, nil
}

func (i *info) rsaKeyDecrypt(b64encrypted string) ([]byte, error) {
	label := []byte{}

	encrypted, err := base64.StdEncoding.DecodeString(b64encrypted)
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
	dec, err := base64.StdEncoding.DecodeString(iv)
	if err != nil {
		return nil, err
	}

	return dec, nil
}
