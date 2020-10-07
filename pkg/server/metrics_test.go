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

	assert.Contains(t, expectedPaths,fmt.Sprintf("/claimed-keys/{startDate:%s}", DATEFORMAT), "Should contain claimed-keys endpoint")
	assert.Contains(t, expectedPaths,fmt.Sprintf("/generated-keys/{startDate:%s}", DATEFORMAT), "Should contain generated-keys endpoint")
	assert.Contains(t, expectedPaths,fmt.Sprintf("/regenerated-keys/{startDate:%s}", DATEFORMAT), "Should contain regenerated-keys endpoint")
	assert.Contains(t, expectedPaths,fmt.Sprintf("/expired-keys/{startDate:%s}", DATEFORMAT), "Should contain expired-keys endpoint")
	assert.Contains(t, expectedPaths,fmt.Sprintf("/exhausted-keys/{startDate:%s}", DATEFORMAT), "Should contain exhausted-keys endpoint")
	assert.Contains(t, expectedPaths,fmt.Sprintf("/unclaimed-keys/{startDate:%s}", DATEFORMAT), "Should contain unclaimed-keys endpoint")

}


func TestMetricsServlet_ClaimedKeys(t *testing.T) {
	tests := []struct {
		event persistence2.EventType
		endPoint string
	}{
		{persistence2.OTKClaimed, "/claimed-keys"},
		{persistence2.OTKGenerated, "/generated-keys"},
		{persistence2.OTKRegenerated, "/regenerated-keys"},
		{persistence2.OTKExhausted, "/exhausted-keys"},
		{persistence2.OTKUnclaimed, "/unclaimed-keys"},
		{persistence2.OTKExpired, "/expired-keys"},
	}

	db, auth := createMocks()
	router := createRouter(db, auth)

	for _, test := range tests {
		db.On("GetServerEventsByType", test.event, "2020-01-01").
			Return(
				[]persistence2.Events{{
					Source: "foo",
					Date: "bar",
					Count: 1,
				}},
				nil,
			)
		resp := runTest(router, test.endPoint)
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "[{\"source\":\"foo\",\"date\":\"bar\",\"count\":1}]",string(resp.Body.Bytes()))
	}
}

func runTest(router *mux.Router, endpoint string) *httptest.ResponseRecorder {


	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/2020-01-01", endpoint), nil)
	req.Header.Set("Authorization", "Basic Zm9vOmJhcg==")

	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	return resp
}
