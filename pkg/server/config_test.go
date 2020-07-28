package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CovidShield/server/pkg/config"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func Router() *mux.Router {
	router := mux.NewRouter()
	return router
}

func GetPaths(router *mux.Router) []string {
	var paths []string
	router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, err := route.GetPathTemplate()
		if err != nil {
			return err
		}
		paths = append(paths, path)
		return nil
	})
	return paths
}

func TestNewConfigServlet(t *testing.T) {
	// Init config
	config.InitConfig()

	expected := &configServlet{}
	assert.Equal(t, expected, NewConfigServlet(), "should return a new configServlet struct")

}

func TestRegisterRoutingConfig(t *testing.T) {

	servlet := NewConfigServlet()
	router := Router()
	servlet.RegisterRouting(router)

	expectedPaths := GetPaths(router)
	assert.Contains(t, expectedPaths, "/exposure-configuration/{region:[\\w]+}.json", "should include an exposure-configuration path")

}

func TestExposureConfig(t *testing.T) {
	servlet := NewConfigServlet()
	router := Router()
	servlet.RegisterRouting(router)

	req, _ := http.NewRequest("GET", "/exposure-configuration/CA.json", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code, "OK response is expected")
	assert.Equal(t, response, string(resp.Body.Bytes()), "Correct response is expected")
}
