package retrieval

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSigner(t *testing.T) {

	os.Setenv("ECDSA_KEY", "")
	assert.PanicsWithValue(t, "no ECDSA_KEY", func() { NewSigner() }, "ECDSA_KEY needs to be defined")

	os.Setenv("ECDSA_KEY", strings.Repeat("z", 242))
	assert.PanicsWithError(t, "encoding/hex: invalid byte: U+007A 'z'", func() { NewSigner() }, "ECDSA_KEY needs to be a valid hex sting")

	os.Setenv("ECDSA_KEY", strings.Repeat("a", 242))
	assert.PanicsWithError(t, "x509: failed to parse EC private key: asn1: structure error: length too large", func() { NewSigner() }, "ECDSA_KEY needs to be a x509 cert")

	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	data, _ := x509.MarshalECPrivateKey(privateKey)
	os.Setenv("ECDSA_KEY", hex.EncodeToString(data))

	expected := &signer{privateKey: privateKey}
	assert.Equal(t, NewSigner(), expected, "should return a signer struct with a private key")

}

func TestSign(t *testing.T) {

	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	publicKey := privateKey.PublicKey
	data, _ := x509.MarshalECPrivateKey(privateKey)
	os.Setenv("ECDSA_KEY", hex.EncodeToString(data))

	signer := NewSigner()

	data = []byte(strings.Repeat("a", 10))
	digest := sha256.Sum256(data)

	receivedSignature, receivedError := signer.Sign(data)

	var esig struct {
		R, S *big.Int
	}
	asn1.Unmarshal(receivedSignature, &esig)

	receivedValidation := ecdsa.Verify(&publicKey, digest[:], esig.R, esig.S)
	expectedValidation := true
	assert.Equal(t, receivedValidation, expectedValidation, "signer should return a valid signature")
	assert.Equal(t, receivedError, nil, "signer should not return an error")
}
