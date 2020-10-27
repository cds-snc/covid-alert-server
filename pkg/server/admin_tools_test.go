package server

import (
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

	servlet := NewAdminToolsServlet(db, auth)
	router := Router()
	servlet.RegisterRouting(router)
	return router
}

func TestNewAdminToolsServlet(t *testing.T) {
	db := &persistence.Conn{}
	auth := &keyclaim.Authenticator{}

	expected := &adminToolsServlet{
		db:   db,
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
	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	// Bad auth token
	req, _ := http.NewRequest("POST", "/cleanDiagnosisKeys", nil)
	req.Header.Set("Authorization", "Bearer badtoken")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	testhelpers.AssertLog(t, hook, 1, logrus.InfoLevel, "bad auth header")
}

func TestAdminToolsServlet_NoAuthHeader(t *testing.T) {
	os.Setenv("ENABLE_TEST_TOOLS", "true")

	db := &persistence.Conn{}

	auth := &keyclaim.Authenticator{}
	auth.On("RegionFromAuthHeader", "").Return("", "", false)

	router := buildAdminToolsServletRouter(db, auth)
	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	req, _ := http.NewRequest("POST", "/cleanDiagnosisKeys", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	testhelpers.AssertLog(t, hook, 1, logrus.InfoLevel, "bad auth header")
}

func TestAdminToolsServlet_GET(t *testing.T) {

	os.Setenv("ENABLE_TEST_TOOLS", "true")
	db := &persistence.Conn{}
	auth := &keyclaim.Authenticator{}

	router := buildAdminToolsServletRouter(db, auth)
	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	req, _ := http.NewRequest("GET", "/cleanDiagnosisKeys", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	testhelpers.AssertLog(t, hook, 1, logrus.InfoLevel, "disallowed method")
}

func TestAdminToolsServlet_CleanDiagnosisKeys(t *testing.T) {
	os.Setenv("ENABLE_TEST_TOOLS", "true")

	db := &persistence.Conn{}
	db.On("ClearDiagnosisKeys", mock.Anything).Return(nil)

	auth := &keyclaim.Authenticator{}
	auth.On("RegionFromAuthHeader", "Bearer goodtoken").Return("", "", true)

	router := buildAdminToolsServletRouter(db, auth)
	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	req, _ := http.NewRequest("POST", "/cleanDiagnosisKeys", nil)
	req.Header.Set("Authorization", "Bearer goodtoken")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code, "OK status expected")
	assert.Equal(t, "cleared diagnosis_keys", string(resp.Body.Bytes()), "Correct response is expected")

	testhelpers.AssertLog(t, hook, 1, logrus.InfoLevel, "cleared diagnosis_keys")

}
