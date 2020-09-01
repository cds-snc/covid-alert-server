package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	persistence "github.com/CovidShield/server/mocks/pkg/persistence"
	"github.com/Shopify/goose/logger"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestNewEventServlet(t *testing.T) {
	cache := &persistence.RedisConn{}

	expected := &eventServlet{
		cache: cache,
	}
	assert.Equal(t, expected, NewEventServlet(cache), "should return a new eventServlet struct")
}

func TestRegisterRoutingEvent(t *testing.T) {
	cache := &persistence.RedisConn{}

	servlet := NewEventServlet(cache)
	router := Router()
	servlet.RegisterRouting(router)

	expectedPaths := GetPaths(router)
	assert.Contains(t, expectedPaths, "/event/nonce", "should include an /event/nonce path")
}

func TestEventNonce(t *testing.T) {
	cache := &persistence.RedisConn{}

	response := "6nrR5kyc/jmLWYs1++4wQ0D6jamcsKCh"

	cache.On("GenerateNonce").Return(response, nil)
	servlet := NewEventServlet(cache)
	router := Router()
	servlet.RegisterRouting(router)

	req, _ := http.NewRequest("POST", "/event/nonce", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code, "OK response is expected")
	assert.Equal(t, response, string(resp.Body.Bytes()), "Correct response is expected")
}

func TestEventNonceError(t *testing.T) {
	// Capture logs
	oldLog := log
	defer func() { log = oldLog }()

	nullLog, hook := test.NewNullLogger()
	nullLog.ExitFunc = func(code int) {}

	log = func(ctx logger.Valuer, err ...error) *logrus.Entry {
		return logrus.NewEntry(nullLog)
	}

	cache := &persistence.RedisConn{}

	response := ""

	cache.On("GenerateNonce").Return(response, fmt.Errorf("error"))
	servlet := NewEventServlet(cache)
	router := Router()
	servlet.RegisterRouting(router)

	req, _ := http.NewRequest("POST", "/event/nonce", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 500, resp.Code, "OK response is expected")
	assert.Equal(t, "server error\n", string(resp.Body.Bytes()), "Server error response is expected")
	assertLog(t, hook, 1, logrus.ErrorLevel, "error constructing nonce")
}
