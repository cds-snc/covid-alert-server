package retrieval

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Shopify/goose/logger"
	"github.com/cds-snc/covid-alert-server/pkg/config"
	"github.com/cds-snc/covid-alert-server/pkg/timemath"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestNewAuthenticator(t *testing.T) {
	// Init config
	config.InitConfig()

	// Capture logs
	oldLog := log
	defer func() { log = oldLog }()

	nullLog, hook := test.NewNullLogger()
	nullLog.ExitFunc = func(code int) {}

	log = func(ctx logger.Valuer, err ...error) *logrus.Entry {
		return logrus.NewEntry(nullLog)
	}

	// No HMAC
	os.Setenv("RETRIEVE_HMAC_KEY", "")
	NewAuthenticator()

	assertLog(t, hook, 1, logrus.FatalLevel, "RETRIEVE_HMAC_KEY missing or too short")

	// Short HMAC
	os.Setenv("RETRIEVE_HMAC_KEY", strings.Repeat("a", (config.AppConstants.HmacKeyLength*2)-2))
	NewAuthenticator()

	assertLog(t, hook, 1, logrus.FatalLevel, "RETRIEVE_HMAC_KEY missing or too short")

	// Invalid HEX HMAC
	os.Setenv("RETRIEVE_HMAC_KEY", strings.Repeat("z", config.AppConstants.HmacKeyLength*2))
	NewAuthenticator()

	assertLog(t, hook, 1, logrus.FatalLevel, "RETRIEVE_HMAC_KEY hex decode failed")

	hmac := strings.Repeat("a", config.AppConstants.HmacKeyLength*2)
	os.Setenv("RETRIEVE_HMAC_KEY", hmac)
	hmacKey := make([]byte, hex.DecodedLen(len(hmac)))
	hex.Decode(hmacKey, []byte(hmac))
	expected := &authenticator{hmacKey: hmacKey}
	assert.Equal(t, NewAuthenticator(), expected, "should return a authenticator struct with a HMAC")
}

func TestAuthenticate(t *testing.T) {
	validHmacKey := strings.Repeat("a", config.AppConstants.HmacKeyLength*2)

	os.Setenv("RETRIEVE_HMAC_KEY", validHmacKey)
	authenticator := NewAuthenticator()

	validRegion := "302"
	validDay := "18444"
	validAuth := "448d9bfd238b34323b70175b4b385fb39d59186711049c6766fa7c890de33a12"

	assert.False(t, authenticator.Authenticate("", validDay, validAuth), "region must be three characters long")
	assert.False(t, authenticator.Authenticate(validRegion, "", validAuth), "day must be five characters long")
	assert.False(t, authenticator.Authenticate(validRegion, validDay, ""), "auth must be 64 characters long")

	invalidAuth := strings.Repeat("z", 64)
	assert.False(t, authenticator.Authenticate(validRegion, validDay, invalidAuth), "auth must be valid hex string")

	hmacKey := make([]byte, hex.DecodedLen(len(validHmacKey)))
	hex.Decode(hmacKey, []byte(validHmacKey))

	currentHour := int(timemath.HourNumber(time.Now()))
	validMessage := validRegion + ":" + validDay + ":" + strconv.Itoa(currentHour)
	mac := hmac.New(sha256.New, []byte(hmacKey))
	mac.Write([]byte(validMessage))
	validAuth = hex.EncodeToString(mac.Sum(nil))

	assert.True(t, authenticator.Authenticate(validRegion, validDay, validAuth), "should return true on valid signature for current hour")

	validMessage = validRegion + ":" + validDay + ":" + strconv.Itoa(currentHour-1)
	mac = hmac.New(sha256.New, []byte(hmacKey))
	mac.Write([]byte(validMessage))
	validAuth = hex.EncodeToString(mac.Sum(nil))

	assert.True(t, authenticator.Authenticate(validRegion, validDay, validAuth), "should return true on valid signature for one hour past")

	validMessage = validRegion + ":" + validDay + ":" + strconv.Itoa(currentHour+1)
	mac = hmac.New(sha256.New, []byte(hmacKey))
	mac.Write([]byte(validMessage))
	validAuth = hex.EncodeToString(mac.Sum(nil))

	assert.True(t, authenticator.Authenticate(validRegion, validDay, validAuth), "should return true on valid signature for one hour future")

	validMessage = validRegion + ":" + validDay + ":" + strconv.Itoa(currentHour-2)
	mac = hmac.New(sha256.New, []byte(hmacKey))
	mac.Write([]byte(validMessage))
	validAuth = hex.EncodeToString(mac.Sum(nil))

	assert.False(t, authenticator.Authenticate(validRegion, validDay, validAuth), "should return false on valid signature for two hours past")

	validMessage = validRegion + ":" + validDay + ":" + strconv.Itoa(currentHour+2)
	mac = hmac.New(sha256.New, []byte(hmacKey))
	mac.Write([]byte(validMessage))
	validAuth = hex.EncodeToString(mac.Sum(nil))

	assert.False(t, authenticator.Authenticate(validRegion, validDay, validAuth), "should return false on valid signature for two hours future")

}

func TestValidMAC(t *testing.T) {
	validMessage := []byte("Lavender's blue, dilly, dilly")
	validKey := []byte(strings.Repeat("a", config.AppConstants.HmacKeyLength*2))
	mac := hmac.New(sha256.New, validKey)
	mac.Write(validMessage)
	validMac := mac.Sum(nil)

	assert.True(t, validMAC(validMessage, validMac, validKey), "should return true on match")

	invalidMessage := []byte("Lavender's green")

	assert.False(t, validMAC(invalidMessage, validMac, validKey), "should return false on no match")
}

func assertLog(t *testing.T, hook *test.Hook, length int, level logrus.Level, msg string) {
	assert.Equal(t, length, len(hook.Entries))
	assert.Equal(t, level, hook.LastEntry().Level)
	assert.Equal(t, msg, hook.LastEntry().Message)
	hook.Reset()
}
