package server

import (
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	keyclaim "github.com/cds-snc/covid-alert-server/mocks/pkg/keyclaim"
	persistence "github.com/cds-snc/covid-alert-server/mocks/pkg/persistence"
)

func buildAdminToolsServletRouter(db *persistence.Conn, auth *keyclaim.Authenticator) *mux.Router {

	servlet := NewAdminToolsServlet(db, auth)
	router := Router()
	servlet.RegisterRouting(router)
	return router
}

func TestNewAdminToolsServlet(t *testing.T) {
	db := &persistence.Conn{}
	auth := &keyclaim.Authenticator{}

	expected := &adminToolsServlet{
		db: db,
		auth: auth,
	}

	assert.Equal(t, expected, NewAdminToolsServlet(db, auth), "should return a new adminToolsServlet struct")
}

func TestAdminToolsServlet_RegisterRouting(t *testing.T) {
	os.Setenv("ENABLE_TEST_TOOLS", "true")
	servlet := NewAdminToolsServlet(&persistence.Conn{}, &keyclaim.Authenticator{})
	router := Router()
	servlet.RegisterRouting(router)

	expectedPaths := GetPaths(router)

	assert.Contains(t, expectedPaths, "/cleanDiagnosisKeys", "should include a /cleanDiagnosisKeys path")
}

func TestAdminToolsServlet_BadAuthToken(t *testing.T) {
	os.Setenv("ENABLE_TEST_TOOLS", "true")
	auth := &keyclaim.Authenticator{}
	// Auth Mock
	auth.On("RegionFromAuthHeader", "Bearer badtoken").Return("", "", false)

	db := &persistence.Conn{}
	router := buildAdminToolsServletRouter(db, auth)
	hook, oldLog := setupTestLogging()
	defer func() { log = oldLog }()

	// Bad auth token
	req, _ := http.NewRequest("POST", "/cleanDiagnosisKeys", nil)
	req.Header.Set("Authorization", "Bearer badtoken")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	assertLog(t, hook, 1, logrus.InfoLevel, "bad auth header")
}

func TestAdminToolsServlet_NoAuthHeader(t *testing.T) {
	os.Setenv("ENABLE_TEST_TOOLS", "true")
	db := &persistence.Conn{}
	auth := &keyclaim.Authenticator{}

	auth.On("RegionFromAuthHeader", "").Return("", "", false)

	router := buildAdminToolsServletRouter(db, auth)
	hook, oldLog := setupTestLogging()
	defer func() { log = oldLog }()

	req, _ := http.NewRequest("POST", "/cleanDiagnosisKeys", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	assertLog(t, hook, 1, logrus.InfoLevel, "bad auth header")
}

func TestAdminToolsServlet_GET(t *testing.T) {

	os.Setenv("ENABLE_TEST_TOOLS", "true")
	db := &persistence.Conn{}
	auth := &keyclaim.Authenticator{}

	router := buildAdminToolsServletRouter(db, auth)
	hook, oldLog := setupTestLogging()
	defer func() { log = oldLog }()

	req, _ := http.NewRequest("GET", "/cleanDiagnosisKeys", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	assertLog(t, hook, 1, logrus.InfoLevel, "disallowed method")
}
