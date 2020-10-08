package server

import (
	"fmt"
	keyclaim "github.com/cds-snc/covid-alert-server/mocks/pkg/keyclaim"
	persistence "github.com/cds-snc/covid-alert-server/mocks/pkg/persistence"
	persistence2 "github.com/cds-snc/covid-alert-server/pkg/persistence"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)


func createRouter(db *persistence.Conn, auth *keyclaim.Authenticator) *mux.Router {

	servlet  := NewMetricsServlet(db, auth)
	router := Router()
	servlet.RegisterRouting(router)

	return router

}

func createMocks() (*persistence.Conn, *keyclaim.Authenticator) {
	return &persistence.Conn{}	, &keyclaim.Authenticator{}
}

func TestNewMetricsServlet(t *testing.T) {

	db, auth := createMocks()

	expected := &metricsServlet{
		db: db,
		auth: auth,
	}

	assert.Equal(t, expected, NewMetricsServlet(db, auth), "should return a new metrics servlet")
}

func TestRegisterRoutingMetrics(t *testing.T) {

	db, auth := createMocks()
	router := createRouter(db, auth)

	expectedPaths := GetPaths(router)

	assert.Contains(t, expectedPaths,fmt.Sprintf("/events/{startDate:%s}", DATEFORMAT), "Should contain claimed-keys endpoint")
}

func TestMetricsServlet_ClaimedKeys(t *testing.T) {

	db, auth := createMocks()
	router := createRouter(db, auth)

		db.On("GetServerEvents",  "2020-01-01").
			Return(
				[]persistence2.Events{{
					Identifier: "event",
					Source: "foo",
					Date: "bar",
					Count: 1,
				}},
				nil,
			)

		req, _ := http.NewRequest("GET", "/events/2020-01-01", nil)
		req.Header.Set("Authorization", "Basic Zm9vOmJhcg==")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "[{\"source\":\"foo\",\"date\":\"bar\",\"count\":1,\"identifier\":\"event\"}]",string(resp.Body.Bytes()))
}
