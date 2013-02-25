package gdssh

// thanks to: http://dave.cheney.net/tag/golang

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/pem"
	"io"
	"io/ioutil"
	"log"
)

type keyring struct {
	keys []*rsa.PrivateKey
}

func (k *keyring) Key(i int) (interface{}, error) {
	if i < 0 || i >= len(k.keys) {
		return nil, nil
	}
	return k.keys[i].PublicKey, nil
}

func (k *keyring) Sign(i int, rand io.Reader, data []byte) ([]byte, error) {
	hash := sha1.New()
	hash.Write(data)
	return rsa.SignPKCS1v15(rand, k.keys[i], crypto.SHA1, hash.Sum(nil))
}

func (k *keyring) loadPEM(file string) error {
	pemBytes, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal("Could not load keyfile '", file, "': ", err)
	}

	block, comment := pem.Decode(pemBytes)
	if block == nil {
		log.Fatal("Could not parse keyfile '", file, "' (", comment, "): ", err)
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Fatal("Could not parse keyfile '", file, "': ", err)
	}

	k.keys = append(k.keys, privateKey)
	return nil
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
