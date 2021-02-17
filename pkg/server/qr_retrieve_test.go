package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/cds-snc/covid-alert-server/pkg/testhelpers"
	timestamp "github.com/golang/protobuf/ptypes"
	"github.com/gorilla/mux"

	persistence "github.com/cds-snc/covid-alert-server/mocks/pkg/persistence"
	retrieval "github.com/cds-snc/covid-alert-server/mocks/pkg/retrieval"
	pb "github.com/cds-snc/covid-alert-server/pkg/proto/covidshield"
	"github.com/cds-snc/covid-alert-server/pkg/timemath"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewQrRetrieveServlet(t *testing.T) {

	db := &persistence.Conn{}
	auth := &retrieval.Authenticator{}
	signer := &retrieval.Signer{}

	expected := &qrRetrieveServlet{
		db:     db,
		auth:   auth,
		signer: signer,
	}
	assert.Equal(t, expected, NewQrRetrieveServlet(db, auth, signer), "should return a new qrRetrieveServlet struct")

}

func TestQrRegisterRoutingRetrieve(t *testing.T) {

	servlet := NewQrRetrieveServlet(&persistence.Conn{}, &retrieval.Authenticator{}, &retrieval.Signer{})
	router := Router()
	servlet.RegisterRouting(router)

	expectedPaths := GetPaths(router)
	assert.Contains(t, expectedPaths, "/qr/{region:[0-9]{3}}/{day:[0-9]{5}}/{auth:.*}", "should include a retrieve path")

}

func TestQrRetrieve_BadAuthmock(t *testing.T) {
	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	db, auth, signer := setupQrRetrieveMockers()

	router := setupQrRetrieveRouter(db, auth, signer)

	badAuth := "dcba"
	region := "302"
	currentDateNumber := fmt.Sprint(timemath.CurrentDateNumber())

	// Bad Auth Mock
	auth.On("Authenticate", region, currentDateNumber, badAuth).Return(false)

	// Bad authentication
	req, _ := http.NewRequest("GET", fmt.Sprintf("/qr/%s/%s/%s", region, currentDateNumber, badAuth), nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 401, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	testhelpers.AssertLog(t, hook, 3, logrus.WarnLevel, "invalid auth parameter")
}

func TestQrRetrieve_GoodAuthmock(t *testing.T) {
	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	db, auth, signer := setupQrRetrieveMockers()
	router := setupQrRetrieveRouter(db, auth, signer)

	goodAuth := "abcd"
	region := "302"
	currentDateNumber := fmt.Sprint(timemath.CurrentDateNumber())

	// Good auth Mock
	auth.On("Authenticate", region, currentDateNumber, goodAuth).Return(true)

	// Bad Method
	req, _ := http.NewRequest("POST", fmt.Sprintf("/qr/%s/%s/%s", region, currentDateNumber, goodAuth), nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 405, resp.Code, "Method not allowed response is expected")
	assert.Equal(t, "method not allowed\n", string(resp.Body.Bytes()), "Correct response is expected")

	testhelpers.AssertLog(t, hook, 3, logrus.WarnLevel, "method not allowed")
}

func TestQrRetrieve_AllKeysDownload(t *testing.T) {
	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	db, auth, signer := setupQrRetrieveMockers()
	router := setupQrRetrieveRouter(db, auth, signer)

	region := "302"
	goodAuth := "abcd"

	// All keys download Mock
	auth.On("Authenticate", region, "00000", goodAuth).Return(true)
	endDate := timemath.CurrentDateNumber() - 1
	startDate := endDate - numberOfDaysToServe

	startTime := time.Unix(int64(startDate*86400), 0)
	endTime := time.Unix(int64((endDate+1)*86400), 0)

	db.On("FetchOutbreakForTimeRange", startTime, endTime).Return([]*pb.OutbreakEvent{randomTestOutbreakEvent(), randomTestOutbreakEvent()}, nil)

	signer.On("Sign", mock.AnythingOfType("[]uint8")).Return(make([]byte, 64), nil)

	// Get all keys for past 14 days
	req, _ := http.NewRequest("GET", fmt.Sprintf("/qr/%s/%s/%s", region, "00000", goodAuth), nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code, "Success response is expected")
	assert.Contains(t, resp.Header()["Content-Type"], "application/zip", "Cache-Control should be set to application/zip")
	assert.Contains(t, resp.Header()["Cache-Control"], "public, max-age=3600, max-stale=600", "Cache-Control should be set to public, max-age=3600, max-stale=600")

	testhelpers.AssertLog(t, hook, 3, logrus.InfoLevel, "Wrote outbreak event retrieval")
}

func TestQrRetrieve_FutureDate(t *testing.T) {

	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	db, auth, signer := setupQrRetrieveMockers()
	router := setupQrRetrieveRouter(db, auth, signer)

	region := "302"
	goodAuth := "abcd"
	futureDate := fmt.Sprint(timemath.CurrentDateNumber() + 1)

	// Future date mock
	auth.On("Authenticate", region, futureDate, goodAuth).Return(true)

	// Get future keys
	req, _ := http.NewRequest("GET", fmt.Sprintf("/qr/%s/%s/%s", region, futureDate, goodAuth), nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 404, resp.Code, "404 response is expected")

	testhelpers.AssertLog(t, hook, 3, logrus.WarnLevel, "request for future data")
}

func TestQrRetrieve_OldDate(t *testing.T) {

	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	db, auth, signer := setupQrRetrieveMockers()
	router := setupQrRetrieveRouter(db, auth, signer)

	region := "302"
	goodAuth := "abcd"
	tooOldDate := fmt.Sprint(timemath.CurrentDateNumber() - 15)

	// Too old date mock
	auth.On("Authenticate", region, tooOldDate, goodAuth).Return(true)

	// Get too old keys
	req, _ := http.NewRequest("GET", fmt.Sprintf("/qr/%s/%s/%s", region, tooOldDate, goodAuth), nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 410, resp.Code, "410 response is expected")

	testhelpers.AssertLog(t, hook, 3, logrus.WarnLevel, "request for too-old data")
}

func TestQrRetrieve_FailedDbCall(t *testing.T) {

	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	db, auth, signer := setupQrRetrieveMockers()
	router := setupQrRetrieveRouter(db, auth, signer)

	region := "302"
	goodAuth := "abcd"
	yesterdaysDate := fmt.Sprint(timemath.CurrentDateNumber() - 1)
	dateNumber64, _ := strconv.ParseUint(yesterdaysDate, 10, 32)

	// Failed DB Call
	auth.On("Authenticate", region, yesterdaysDate, goodAuth).Return(true)
	startTime := time.Unix(int64(dateNumber64*86400), 0)
	endTime := time.Unix(int64((dateNumber64+1)*86400), 0)

	db.On("FetchOutbreakForTimeRange", startTime, endTime).Return([]*pb.OutbreakEvent{}, fmt.Errorf("error"))

	// Failing DB message
	req, _ := http.NewRequest("GET", fmt.Sprintf("/qr/%s/%s/%s", region, yesterdaysDate, goodAuth), nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 500, resp.Code, "500 response is expected")

	testhelpers.AssertLog(t, hook, 3, logrus.ErrorLevel, "database error")

}

func setupQrRetrieveMockers() (*persistence.Conn, *retrieval.Authenticator, *retrieval.Signer) {

	db := &persistence.Conn{}
	auth := &retrieval.Authenticator{}
	signer := &retrieval.Signer{}

	return db, auth, signer
}

func setupQrRetrieveRouter(db *persistence.Conn, auth *retrieval.Authenticator, signer *retrieval.Signer) *mux.Router {

	servlet := NewQrRetrieveServlet(db, auth, signer)
	router := Router()
	servlet.RegisterRouting(router)

	return router
}

func randomTestOutbreakEvent() *pb.OutbreakEvent {
	uuid := "8a2c34b2-74a5-4b6a-8bed-79b7823b37c7"
	startTime, _ := timestamp.TimestampProto(time.Unix(1613238163, 0))
	endTime, _ := timestamp.TimestampProto(time.Unix(1613324563, 0))
	location := &pb.OutbreakEvent{LocationId: &uuid, StartTime: startTime, EndTime: endTime}
	return location
}
