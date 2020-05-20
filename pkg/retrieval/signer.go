package retrieval

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
)

type Signer interface {
	Sign([]byte) ([]byte, error)
}

type signer struct {
	privateKey *ecdsa.PrivateKey
}

func NewSigner(key *ecdsa.PrivateKey) Signer {
	return &signer{privateKey: key}
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

func DummyKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}
