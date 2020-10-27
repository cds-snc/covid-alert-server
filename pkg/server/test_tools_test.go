package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	keyclaim "github.com/cds-snc/covid-alert-server/mocks/pkg/keyclaim"
	persistence "github.com/cds-snc/covid-alert-server/mocks/pkg/persistence"
	"github.com/cds-snc/covid-alert-server/pkg/testhelpers"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func buildAdminToolsServletRouter(db *persistence.Conn, auth *keyclaim.Authenticator) *mux.Router {

	servlet := NewTestToolsServlet(db, auth)
	router := Router()
	servlet.RegisterRouting(router)
	return router
}

func TestNewTestToolsServlet(t *testing.T) {
	db := &persistence.Conn{}
	auth := &keyclaim.Authenticator{}

	expected := &testToolsServlet{
		db:   db,
		auth: auth,
	}

	assert.Equal(t, expected, NewTestToolsServlet(db, auth), "should return a new testToolsServlet struct")
}

func TestTestToolsServlet_RegisterRouting(t *testing.T) {
	os.Setenv("ENABLE_TEST_TOOLS", "true")
	servlet := NewTestToolsServlet(&persistence.Conn{}, &keyclaim.Authenticator{})
	router := Router()
	servlet.RegisterRouting(router)

	expectedPaths := GetPaths(router)

	assert.Contains(t, expectedPaths, "/clear-diagnosis-keys", "should include a /clear-diagnosis-keys path")
}

func TestTestToolsServlet_RegisterRoutingDisabled(t *testing.T) {
	os.Setenv("ENABLE_TEST_TOOLS", "false")
	servlet := NewTestToolsServlet(&persistence.Conn{}, &keyclaim.Authenticator{})
	router := Router()
	assert.Panics(t, func () { servlet.RegisterRouting(router) }, "should panic if called while ENABLE_TEST_TOOLS is false")

}

func TestTestToolsServlet_RegisterRoutingProduction(t *testing.T) {
	os.Setenv("ENABLE_TEST_TOOLS", "true")
	os.Setenv("ENV", "production")

	servlet := NewTestToolsServlet(&persistence.Conn{}, &keyclaim.Authenticator{})
	router := Router()
	assert.Panics(t, func () { servlet.RegisterRouting(router) }, "should panic if ENV is production")
	os.Setenv("ENV", "")
}

func TestTestToolsServlet_BadAuthToken(t *testing.T) {
	os.Setenv("ENABLE_TEST_TOOLS", "true")
	auth := &keyclaim.Authenticator{}
	// Auth Mock
	auth.On("RegionFromAuthHeader", "Bearer badtoken").Return("", "", false)

	db := &persistence.Conn{}
	router := buildAdminToolsServletRouter(db, auth)
	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	// Bad auth token
	req, _ := http.NewRequest("POST", "/clear-diagnosis-keys", nil)
	req.Header.Set("Authorization", "Bearer badtoken")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	testhelpers.AssertLog(t, hook, 1, logrus.InfoLevel, "bad auth header")
}

func TestTestToolsServlet_NoAuthHeader(t *testing.T) {
	os.Setenv("ENABLE_TEST_TOOLS", "true")

	db := &persistence.Conn{}

	auth := &keyclaim.Authenticator{}
	auth.On("RegionFromAuthHeader", "").Return("", "", false)

	router := buildAdminToolsServletRouter(db, auth)
	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	req, _ := http.NewRequest("POST", "/clear-diagnosis-keys", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	testhelpers.AssertLog(t, hook, 1, logrus.InfoLevel, "bad auth header")
}

func TestTestToolsServlet_GET(t *testing.T) {

	os.Setenv("ENABLE_TEST_TOOLS", "true")
	db := &persistence.Conn{}
	auth := &keyclaim.Authenticator{}

	router := buildAdminToolsServletRouter(db, auth)
	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	req, _ := http.NewRequest("GET", "/clear-diagnosis-keys", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	testhelpers.AssertLog(t, hook, 1, logrus.InfoLevel, "disallowed method")
}

func TestTestToolsServlet_ClearDiagnosisKeys(t *testing.T) {
	os.Setenv("ENABLE_TEST_TOOLS", "true")

	db := &persistence.Conn{}
	db.On("ClearDiagnosisKeys", mock.Anything).Return(nil)

	auth := &keyclaim.Authenticator{}
	auth.On("RegionFromAuthHeader", "Bearer goodtoken").Return("", "", true)

	router := buildAdminToolsServletRouter(db, auth)
	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	req, _ := http.NewRequest("POST", "/clear-diagnosis-keys", nil)
	req.Header.Set("Authorization", "Bearer goodtoken")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code, "OK status expected")
	assert.Equal(t, "cleared diagnosis_keys", string(resp.Body.Bytes()), "Correct response is expected")

	testhelpers.AssertLog(t, hook, 1, logrus.InfoLevel, "cleared diagnosis_keys")

}

func TestTestToolsServlet_ClearDiagnosisKeysFailed(t *testing.T) {
	os.Setenv("ENABLE_TEST_TOOLS", "true")

	db := &persistence.Conn{}
	db.On("ClearDiagnosisKeys", mock.Anything).Return(fmt.Errorf("oh no"))

	auth := &keyclaim.Authenticator{}
	auth.On("RegionFromAuthHeader", "Bearer goodtoken").Return("", "", true)

	router := buildAdminToolsServletRouter(db, auth)
	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	req, _ := http.NewRequest("POST", "/clear-diagnosis-keys", nil)
	req.Header.Set("Authorization", "Bearer goodtoken")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code, "Internal Server Error Expected")
	assert.Equal(t, "unable to clear diagnosis_keys\n", string(resp.Body.Bytes()), "Correct response is expected")

	testhelpers.AssertLog(t, hook, 1, logrus.ErrorLevel, "unable to clear diagnosis_keys")

}
