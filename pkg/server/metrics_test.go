package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	keyclaim "github.com/cds-snc/covid-alert-server/mocks/pkg/keyclaim"
	persistence "github.com/cds-snc/covid-alert-server/mocks/pkg/persistence"
	persistence2 "github.com/cds-snc/covid-alert-server/pkg/persistence"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func createRouter(db *persistence.Conn, auth *keyclaim.Authenticator) *mux.Router {

	servlet := NewMetricsServlet(db, auth)
	router := Router()
	servlet.RegisterRouting(router)

	return router

}

func createMocks() (*persistence.Conn, *keyclaim.Authenticator) {
	return &persistence.Conn{}, &keyclaim.Authenticator{}
}

func TestNewMetricsServlet(t *testing.T) {

	db, auth := createMocks()

	expected := &metricsServlet{
		db:   db,
		auth: auth,
	}

	assert.Equal(t, expected, NewMetricsServlet(db, auth), "should return a new metrics servlet")
}

func TestMetricsServlet_BasicAuth(t *testing.T) {
	db, auth := createMocks()
	router := createRouter(db, auth)

	req, _ := http.NewRequest("GET", "/events/2020-01-01", nil)
	req.Header.Set("Authorization", "Basic foo")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()))
}

func TestMetricsServlet_InvalidDateFormat(t *testing.T) {
	db, auth := createMocks()
	router := createRouter(db, auth)

	req, _ := http.NewRequest("GET", "/events/01-01-2001", nil)
	req.Header.Set("Authorization", "Basic Zm9vOmJhcg==")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNotFound, resp.Code)

	req, _ = http.NewRequest("GET", "/events/uploads/01-01-2001", nil)
	req.Header.Set("Authorization", "Basic Zm9vOmJhcg==")

	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func TestMetricsServlet_ParseDate(t *testing.T) {
	db, auth := createMocks()
	router := createRouter(db, auth)

	req, _ := http.NewRequest("GET", "/events/2001-32-01", nil)
	req.Header.Set("Authorization", "Basic Zm9vOmJhcg==")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "error parsing date\n", string(resp.Body.Bytes()))

	req, _ = http.NewRequest("GET", "/events/uploads/2001-32-01", nil)
	req.Header.Set("Authorization", "Basic Zm9vOmJhcg==")

	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "error parsing date\n", string(resp.Body.Bytes()))
}

func TestMetricsServlet_DisallowedMethods(t *testing.T) {

	httpVerbs := []string{"POST", "PUT", "DELETE", "PATCH", "OPTIONS"}

	db, auth := createMocks()
	router := createRouter(db, auth)

	for _, verb := range httpVerbs {
		req, _ := http.NewRequest(verb, "/events/2001-01-01", nil)
		req.Header.Set("Authorization", "Basic Zm9vOmJhcg==")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
		assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()))
	}

	for _, verb := range httpVerbs {
		req, _ := http.NewRequest(verb, "/events/uploads/2001-01-01", nil)
		req.Header.Set("Authorization", "Basic Zm9vOmJhcg==")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
		assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()))
	}
}

func TestMetricsServlet_RegisterRoutingMetrics(t *testing.T) {

	db, auth := createMocks()
	router := createRouter(db, auth)

	expectedPaths := GetPaths(router)

	assert.Contains(t, expectedPaths, fmt.Sprintf("/events/{startDate:%s}", DATEFORMAT), "Should contain claimed-keys endpoint")
	assert.Contains(t, expectedPaths, fmt.Sprintf("/events/uploads/{startDate:%s}", DATEFORMAT), "Should contain TEK uploads endpoint")
}

func TestMetricsServlet_DBError(t *testing.T) {

	db, auth := createMocks()
	router := createRouter(db, auth)

	db.On("GetServerEvents", "2020-01-01").
		Return(
			nil,
			fmt.Errorf("error"),
		)

	req, _ := http.NewRequest("GET", "/events/2020-01-01", nil)
	req.Header.Set("Authorization", "Basic Zm9vOmJhcg==")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "error retrieving events\n", string(resp.Body.Bytes()))
}

func TestMetricsServlet_DBErrorUploads(t *testing.T) {

	db, auth := createMocks()
	router := createRouter(db, auth)

	db.On("GetTEKUploads", "2020-01-01").
		Return(
			nil,
			fmt.Errorf("error"),
		)

	req, _ := http.NewRequest("GET", "/events/uploads/2020-01-01", nil)
	req.Header.Set("Authorization", "Basic Zm9vOmJhcg==")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "error retrieving upload events\n", string(resp.Body.Bytes()))
}

func TestMetricsServlet_ClaimedKeys(t *testing.T) {

	db, auth := createMocks()
	router := createRouter(db, auth)

	db.On("GetServerEvents", "2020-01-01").
		Return(
			[]persistence2.Events{{
				Identifier: "event",
				Source:     "foo",
				Date:       "bar",
				Count:      1,
			}},
			nil,
		)

	req, _ := http.NewRequest("GET", "/events/2020-01-01", nil)
	req.Header.Set("Authorization", "Basic Zm9vOmJhcg==")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "[{\"source\":\"foo\",\"date\":\"bar\",\"count\":1,\"identifier\":\"event\"}]", string(resp.Body.Bytes()))
}

func TestMetricsServlet_GetTEKUploadsData(t *testing.T) {

	db, auth := createMocks()
	router := createRouter(db, auth)

	db.On("GetTEKUploads", "2020-01-01").
		Return(
			[]persistence2.Uploads{{
				Source:      "foo",
				Date:        "bar",
				Count:       1,
				FirstUpload: true,
			}, {
				Source:      "foo",
				Date:        "bar",
				Count:       1,
				FirstUpload: false,
			}},
			nil,
		)

	req, _ := http.NewRequest("GET", "/events/uploads/2020-01-01", nil)
	req.Header.Set("Authorization", "Basic Zm9vOmJhcg==")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "[{\"source\":\"foo\",\"date\":\"bar\",\"count\":1,\"first_upload\":true},{\"source\":\"foo\",\"date\":\"bar\",\"count\":1,\"first_upload\":false}]", string(resp.Body.Bytes()))
}
