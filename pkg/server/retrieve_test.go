package server

import (
	"crypto/rand"
	"fmt"
	"github.com/cds-snc/covid-alert-server/pkg/testhelpers"
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"testing"

	persistence "github.com/cds-snc/covid-alert-server/mocks/pkg/persistence"
	retrieval "github.com/cds-snc/covid-alert-server/mocks/pkg/retrieval"
	pb "github.com/cds-snc/covid-alert-server/pkg/proto/covidshield"
	"github.com/cds-snc/covid-alert-server/pkg/timemath"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewRetrieveServlet(t *testing.T) {

	db := &persistence.Conn{}
	auth := &retrieval.Authenticator{}
	signer := &retrieval.Signer{}

	expected := &retrieveServlet{
		db:     db,
		auth:   auth,
		signer: signer,
	}
	assert.Equal(t, expected, NewRetrieveServlet(db, auth, signer), "should return a new retrieveServlet struct")

}

func TestRegisterRoutingRetrieve(t *testing.T) {

	servlet := NewRetrieveServlet(&persistence.Conn{}, &retrieval.Authenticator{}, &retrieval.Signer{})
	router := Router()
	servlet.RegisterRouting(router)

	expectedPaths := GetPaths(router)
	assert.Contains(t, expectedPaths, "/retrieve/{region:[0-9]{3}}/{day:[0-9]{5}}/{auth:.*}", "should include a retrieve path")

}

func setupRetrieveMockers() (*persistence.Conn, *retrieval.Authenticator, *retrieval.Signer) {

	db := &persistence.Conn{}
	auth := &retrieval.Authenticator{}
	signer := &retrieval.Signer{}

	return db, auth, signer
}

func setupRouter(db *persistence.Conn, auth *retrieval.Authenticator, signer *retrieval.Signer) *mux.Router {

	servlet := NewRetrieveServlet(db, auth, signer)
	router := Router()
	servlet.RegisterRouting(router)

	return router
}

func TestRetrieve_BadAuthmock(t *testing.T) {
	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	db, auth, signer := setupRetrieveMockers()

	router := setupRouter(db, auth, signer)

	badAuth := "dcba"
	region := "302"
	currentDateNumber := fmt.Sprint(timemath.CurrentDateNumber())

	// Bad Auth Mock
	auth.On("Authenticate", region, currentDateNumber, badAuth).Return(false)

	// Bad authentication
	req, _ := http.NewRequest("GET", fmt.Sprintf("/retrieve/%s/%s/%s", region, currentDateNumber, badAuth), nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 401, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	testhelpers.AssertLog(t, hook, 1, logrus.WarnLevel, "invalid auth parameter")
}

func TestRetrieve_GoodAuthmock(t *testing.T) {
	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	db, auth, signer := setupRetrieveMockers()
	router := setupRouter(db, auth, signer)

	goodAuth := "abcd"
	region := "302"
	currentDateNumber := fmt.Sprint(timemath.CurrentDateNumber())

	// Good auth Mock
	auth.On("Authenticate", region, currentDateNumber, goodAuth).Return(true)

	// Bad Method
	req, _ := http.NewRequest("POST", fmt.Sprintf("/retrieve/%s/%s/%s", region, currentDateNumber, goodAuth), nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 405, resp.Code, "Method not allowed response is expected")
	assert.Equal(t, "method not allowed\n", string(resp.Body.Bytes()), "Correct response is expected")

	testhelpers.AssertLog(t, hook, 1, logrus.WarnLevel, "method not allowed")
}

func TestRetrieve_AllKeysDownload(t *testing.T) {
	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	db, auth, signer := setupRetrieveMockers()
	router := setupRouter(db, auth, signer)

	region := "302"
	goodAuth := "abcd"
	currentRSIN := pb.CurrentRollingStartIntervalNumber()

	// All keys download Mock
	auth.On("Authenticate", region, "00000", goodAuth).Return(true)
	startHour := (timemath.CurrentDateNumber() - 15) * 24
	endHour := timemath.CurrentDateNumber() * 24

	db.On("FetchKeysForHours", region, startHour, endHour, currentRSIN).Return([]*pb.TemporaryExposureKey{randomTestKey(), randomTestKey()}, nil)

	signer.On("Sign", mock.AnythingOfType("[]uint8")).Return(make([]byte, 64), nil)

	// Get all keys for past 14 days
	req, _ := http.NewRequest("GET", fmt.Sprintf("/retrieve/%s/%s/%s", region, "00000", goodAuth), nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code, "Success response is expected")
	assert.Contains(t, resp.Header()["Content-Type"], "application/zip", "Cache-Control should be set to application/zip")
	assert.Contains(t, resp.Header()["Cache-Control"], "public, max-age=3600, max-stale=600", "Cache-Control should be set to public, max-age=3600, max-stale=600")

	testhelpers.AssertLog(t, hook, 1, logrus.InfoLevel, "Wrote retrieval")
}
func TestRetrieve_FutureDate(t *testing.T) {

	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	db, auth, signer := setupRetrieveMockers()
	router := setupRouter(db, auth, signer)

	region := "302"
	goodAuth := "abcd"
	futureDate := fmt.Sprint(timemath.CurrentDateNumber() + 1)

	// Future date mock
	auth.On("Authenticate", region, futureDate, goodAuth).Return(true)

	// Get future keys
	req, _ := http.NewRequest("GET", fmt.Sprintf("/retrieve/%s/%s/%s", region, futureDate, goodAuth), nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 404, resp.Code, "404 response is expected")

	testhelpers.AssertLog(t, hook, 1, logrus.WarnLevel, "request for future data")
}

func TestRetrieve_OldDate(t *testing.T) {

	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	db, auth, signer := setupRetrieveMockers()
	router := setupRouter(db, auth, signer)

	region := "302"
	goodAuth := "abcd"
	tooOldDate := fmt.Sprint(timemath.CurrentDateNumber() - 15)

	// Too old date mock
	auth.On("Authenticate", region, tooOldDate, goodAuth).Return(true)

	// Get too old keys
	req, _ := http.NewRequest("GET", fmt.Sprintf("/retrieve/%s/%s/%s", region, tooOldDate, goodAuth), nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 410, resp.Code, "410 response is expected")

	testhelpers.AssertLog(t, hook, 1, logrus.WarnLevel, "request for too-old data")
}

func TestRetrieve(t *testing.T) {

	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	db, auth, signer := setupRetrieveMockers()
	router := setupRouter(db, auth, signer)

	region := "302"
	goodAuth := "abcd"
	currentRSIN := pb.CurrentRollingStartIntervalNumber()
	yesterdaysDate := fmt.Sprint(timemath.CurrentDateNumber() - 1)

	// Failed DB Call
	auth.On("Authenticate", region, yesterdaysDate, goodAuth).Return(true)
	startHour := (timemath.CurrentDateNumber() - 1) * 24
	endHour := timemath.CurrentDateNumber() * 24

	db.On("FetchKeysForHours", region, startHour, endHour, currentRSIN).Return([]*pb.TemporaryExposureKey{}, fmt.Errorf("error"))

	// Failing DB message
	req, _ := http.NewRequest("GET", fmt.Sprintf("/retrieve/%s/%s/%s", region, yesterdaysDate, goodAuth), nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 500, resp.Code, "500 response is expected")

	testhelpers.AssertLog(t, hook, 1, logrus.ErrorLevel, "database error")

}

func randomTestKey() *pb.TemporaryExposureKey {
	token := make([]byte, 16)
	rand.Read(token)
	transmissionRiskLevel := int32(2)
	rollingStartIntervalNumber := int32(2651450)
	rollingPeriod := int32(144)
	key := &pb.TemporaryExposureKey{
		KeyData:                    token,
		TransmissionRiskLevel:      &transmissionRiskLevel,
		RollingStartIntervalNumber: &rollingStartIntervalNumber,
		RollingPeriod:              &rollingPeriod,
	}
	return key
}
