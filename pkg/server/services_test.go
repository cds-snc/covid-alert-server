package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"testing"

	"github.com/Shopify/goose/srvutil"
	"github.com/stretchr/testify/assert"
)

func TestNewServicesServlet(t *testing.T) {
	s := &servicesServlet{}
	// Compare function names vs. functions
	funcName1 := runtime.FuncForPC(reflect.ValueOf(srvutil.PrefixServlet(s, "/services")).Pointer()).Name()
	funcName2 := runtime.FuncForPC(reflect.ValueOf(NewServicesServlet()).Pointer()).Name()
	assert.Equal(t, funcName1, funcName2, "should return a new servicesServlet function")

}

func TestRegisterRoutingServices(t *testing.T) {

	servlet := NewServicesServlet()
	router := Router()
	servlet.RegisterRouting(router)

	expectedPaths := GetPaths(router)
	assert.Contains(t, expectedPaths, "/services/ping", "should include a ping path")
	assert.Contains(t, expectedPaths, "/services/version.json", "should include a version.json path")
	assert.Contains(t, expectedPaths, "/services/present", "should include a present path")

}

func TestPing(t *testing.T) {
	servlet := NewServicesServlet()
	router := Router()
	servlet.RegisterRouting(router)

	req, _ := http.NewRequest("GET", "/services/ping", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	expected := "OK\n"

	assert.Equal(t, 200, resp.Code, "OK response is expected")
	assert.Equal(t, expected, string(resp.Body.Bytes()), "OK response is expected")
	assert.Contains(t, resp.Header()["Cache-Control"], "no-store", "Cache-Control should be set to no-store")
	assert.Contains(t, resp.Header()["Content-Type"], "text/plain; charset=utf-8", "Cache-Type should be set to text/plain; charset=utf-8")
}

func TestPresent(t *testing.T) {
	servlet := NewServicesServlet()
	router := Router()
	servlet.RegisterRouting(router)

	req, _ := http.NewRequest("GET", "/services/present", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 204, resp.Code, "No content response is expected")
	assert.Contains(t, resp.Header()["Cache-Control"], "no-store", "Cache-Control should be set to no-store")
}

func TestVersion(t *testing.T) {

	branch = "main"
	revision = "abcd"

	servlet := NewServicesServlet()
	router := Router()
	servlet.RegisterRouting(router)

	req, _ := http.NewRequest("GET", "/services/version.json", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	version := version{
		Branch:   branch,
		Revision: revision,
	}

	expected, _ := json.Marshal(version)

	assert.Equal(t, 200, resp.Code, "OK response is expected")
	assert.Equal(t, string(expected), string(resp.Body.Bytes()), "JSON response is expected")
	assert.Contains(t, resp.Header()["Cache-Control"], "no-store", "Cache-Control should be set to no-store")
	assert.Contains(t, resp.Header()["Content-Type"], "application/json; charset=utf-8", "Cache-Type should be set to application/json; charset=utf-8")
}
