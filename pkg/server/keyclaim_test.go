package server

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	keyclaim "github.com/CovidShield/server/mocks/pkg/keyclaim"
	persistence "github.com/CovidShield/server/mocks/pkg/persistence"
	"github.com/CovidShield/server/pkg/config"
	err "github.com/CovidShield/server/pkg/persistence"
	"github.com/Shopify/goose/logger"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/stretchr/testify/assert"
)

func TestNewKeyClaimServlet(t *testing.T) {
	db := &persistence.Conn{}
	auth := &keyclaim.Authenticator{}

	expected := &keyClaimServlet{
		db:   db,
		auth: auth,
	}
	assert.Equal(t, expected, NewKeyClaimServlet(db, auth), "should return a new keyClaimServlet struct")
}

func TestRegisterRoutingKeyClaim(t *testing.T) {
	servlet := NewKeyClaimServlet(&persistence.Conn{}, &keyclaim.Authenticator{})
	router := Router()
	servlet.RegisterRouting(router)

	expectedPaths := GetPaths(router)
	assert.Contains(t, expectedPaths, "/new-key-claim", "should include a /new-key-claim path")
	assert.Contains(t, expectedPaths, "/new-key-claim/{hashID:[0-9,a-z]{128}}", "should include a /new-key-claim/{hashID:[0-9,a-z]{128}} path")
	assert.Contains(t, expectedPaths, "/claim-key", "should include a claim-key path")
}

func TestNewKeyClaim(t *testing.T) {
	db := &persistence.Conn{}
	auth := &keyclaim.Authenticator{}

	// Auth Mock
	auth.On("Authenticate", "badtoken").Return("", false)
	auth.On("Authenticate", "goodtoken").Return("302", true)
	auth.On("Authenticate", "errortoken").Return("302", true)

	hashID := hex.EncodeToString(SHA512([]byte("abcd")))

	// DB Mock
	db.On("NewKeyClaim", "302", "goodtoken", "").Return("AAABBBCCCC", nil)
	db.On("NewKeyClaim", "302", "goodtoken", hashID).Return("AAABBBCCCC", nil)

	db.On("NewKeyClaim", "302", "errortoken", "").Return("", fmt.Errorf("Random error"))
	db.On("NewKeyClaim", "302", "errortoken", hashID).Return("", err.ErrHashIDClaimed)

	servlet := NewKeyClaimServlet(db, auth)
	router := Router()
	servlet.RegisterRouting(router)

	// Capture logs
	oldLog := log
	defer func() { log = oldLog }()

	nullLog, hook := test.NewNullLogger()
	nullLog.ExitFunc = func(code int) {}

	log = func(ctx logger.Valuer, err ...error) *logrus.Entry {
		return logrus.NewEntry(nullLog)
	}

	// Return CORS options header
	req, _ := http.NewRequest("OPTIONS", "/new-key-claim", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code, "OK response is expected")
	assert.Contains(t, resp.Header()["Access-Control-Allow-Origin"], config.AppConstants.CORSAccessControlAllowOrigin, "Access-Control-Allow-Origin should be set to the config value")
	assert.Contains(t, resp.Header()["Access-Control-Allow-Methods"], "POST", "Access-Control-Allow-Methods should be set to POST")
	assert.Contains(t, resp.Header()["Access-Control-Allow-Headers"], "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Referer, User-Agent", "Access-Control-Allow-Headers should be set to Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Referer, User-Agent")

	// No a POST request
	req, _ = http.NewRequest("GET", "/new-key-claim", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 401, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.InfoLevel, hook.LastEntry().Level)
	assert.Equal(t, "disallowed method", hook.LastEntry().Message)
	hook.Reset()

	// No auth header
	req, _ = http.NewRequest("POST", "/new-key-claim", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 401, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.InfoLevel, hook.LastEntry().Level)
	assert.Equal(t, "bad auth header", hook.LastEntry().Message)
	hook.Reset()

	// Malformed auth header - Bear
	req, _ = http.NewRequest("POST", "/new-key-claim", nil)
	req.Header.Set("Authorization", "Bear thisisaverylongtoken")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 401, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.InfoLevel, hook.LastEntry().Level)
	assert.Equal(t, "bad auth header", hook.LastEntry().Message)
	hook.Reset()

	// Malformed auth header - No space
	req, _ = http.NewRequest("POST", "/new-key-claim", nil)
	req.Header.Set("Authorization", "Bearerthisisaverylongtoken")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 401, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.InfoLevel, hook.LastEntry().Level)
	assert.Equal(t, "bad auth header", hook.LastEntry().Message)
	hook.Reset()

	// Bad auth token
	req, _ = http.NewRequest("POST", "/new-key-claim", nil)
	req.Header.Set("Authorization", "Bearer badtoken")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 401, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.InfoLevel, hook.LastEntry().Level)
	assert.Equal(t, "bad auth header", hook.LastEntry().Message)
	hook.Reset()

	// Good auth token - no HashID
	req, _ = http.NewRequest("POST", "/new-key-claim", nil)
	req.Header.Set("Authorization", "Bearer goodtoken")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code, "Success response is expected")
	assert.Equal(t, "AAABBBCCCC\n", string(resp.Body.Bytes()), "Correct response is expected")

	// Good auth token -  HashID
	req, _ = http.NewRequest("POST", "/new-key-claim/"+hashID, nil)
	req.Header.Set("Authorization", "Bearer goodtoken")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code, "Success response is expected")
	assert.Equal(t, "AAABBBCCCC\n", string(resp.Body.Bytes()), "Correct response is expected")

	// Error saving - no HashID
	req, _ = http.NewRequest("POST", "/new-key-claim", nil)
	req.Header.Set("Authorization", "Bearer errortoken")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 500, resp.Code, "Server error response is expected")
	assert.Equal(t, "server error\n", string(resp.Body.Bytes()), "server error response is expected")

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.ErrorLevel, hook.LastEntry().Level)
	assert.Equal(t, "error constructing new key claim", hook.LastEntry().Message)
	hook.Reset()

	// Error saving - duplicate HashID
	req, _ = http.NewRequest("POST", "/new-key-claim/"+hashID, nil)
	req.Header.Set("Authorization", "Bearer errortoken")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 403, resp.Code, "forbidden response is expected")
	assert.Equal(t, "forbidden\n", string(resp.Body.Bytes()), "forbidden response is expected")

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.InfoLevel, hook.LastEntry().Level)
	assert.Equal(t, "hashID used", hook.LastEntry().Message)
	hook.Reset()
}

func SHA512(message []byte) []byte {
	c := sha512.New()
	c.Write(message)
	return c.Sum(nil)
}
