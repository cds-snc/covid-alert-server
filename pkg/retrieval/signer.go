package retrieval

import (
	"crypto"
	"crypto/ecdsa"
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
	digest := sha256.Sum256(data)
	sig, err := s.privateKey.Sign(rand.Reader, digest[:], crypto.SHA256)
	if err != nil {
		return nil, err
	}
	return sig, nil
}
