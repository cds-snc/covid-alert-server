package retrieval

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"os"
)

type Signer interface {
	Sign([]byte) ([]byte, error)
}

type signer struct {
	privateKey *ecdsa.PrivateKey
}

func NewSigner() Signer {
	ecdsaKeyHex := os.Getenv("ECDSA_KEY")
	if ecdsaKeyHex == "" {
		panic("no ECDSA_KEY")
	}
	ecdsaKey, err := hex.DecodeString(ecdsaKeyHex)
	if err != nil {
		panic(err)
	}

	priv, err := x509.ParseECPrivateKey(ecdsaKey)
	if err != nil {
		panic(err)
	}

	return &signer{privateKey: priv}
}

func (s *signer) Sign(data []byte) ([]byte, error) {
	hash := sha256.Sum256(data)
	a, b, err := ecdsa.Sign(rand.Reader, s.privateKey, hash[:])
	if err != nil {
		return nil, err
	}
	signatureX962 := elliptic.Marshal(elliptic.P256(), a, b)
	return signatureX962, nil
}
