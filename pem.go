package raopd

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
)

func GetRSAPrivateKey(r io.Reader) (*rsa.PrivateKey, error) {
	b := make([]byte, 8192)

	n, err := r.Read(b)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(b[0:n])
	if err != nil {
		return nil, err
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return priv, nil
}
