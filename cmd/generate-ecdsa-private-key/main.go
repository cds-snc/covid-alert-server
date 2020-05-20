package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"fmt"
)

func main() {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	data, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		panic(err)
	}
	fmt.Println(hex.EncodeToString(data))
}
