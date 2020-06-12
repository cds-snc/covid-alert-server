package retrieval

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strconv"
	"time"

	"github.com/CovidShield/server/pkg/timemath"
)

type Authenticator interface {
	Authenticate(string, string, string) bool
}

type authenticator struct {
	hmacKey []byte
}

const hmacKeyLength = 32

func NewAuthenticator() Authenticator {
	retrieveHmacKey := os.Getenv("RETRIEVE_HMAC_KEY")
	if len(retrieveHmacKey) < hex.EncodedLen(hmacKeyLength) {
		log(nil, nil).Fatal("RETRIEVE_HMAC_KEY missing or too short")
	}

	hmacKey := make([]byte, hex.DecodedLen(len(retrieveHmacKey)))
	_, err := hex.Decode(hmacKey, []byte(retrieveHmacKey))
	if err != nil {
		log(nil, err).Fatal("RETRIEVE_HMAC_KEY hex decode failed")
	}

	return &authenticator{hmacKey: hmacKey}
}

func (a *authenticator) Authenticate(region, requestedDay, auth string) bool {
	if len(region) != 3 || len(requestedDay) != 5 || len(auth) != 64 {
		return false
	}

	dst := make([]byte, hex.DecodedLen(len(auth)))
	n, err := hex.Decode(dst, []byte(auth))
	if err != nil {
		return false
	}
	if n != hmacKeyLength {
		return false
	}

	currentHour := int(timemath.HourNumber(time.Now()))

	base := region + ":" + requestedDay + ":"

	if validMAC([]byte(base+strconv.Itoa(currentHour)), dst, a.hmacKey) {
		return true
	}
	if validMAC([]byte(base+strconv.Itoa(currentHour-1)), dst, a.hmacKey) {
		return true
	}
	if validMAC([]byte(base+strconv.Itoa(currentHour+1)), dst, a.hmacKey) {
		return true
	}
	return false
}

func validMAC(message, messageMAC, key []byte) bool {
	mac := hmac.New(sha256.New, key)
	if _, err := mac.Write(message); err != nil {
		log(nil, err).Warn("mac.Write error")
		return false
	}
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(messageMAC, expectedMAC)
}
