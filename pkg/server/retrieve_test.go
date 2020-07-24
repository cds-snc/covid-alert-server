package server

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	persistence "github.com/CovidShield/server/mocks/pkg/persistence"
	retrieval "github.com/CovidShield/server/mocks/pkg/retrieval"
	"github.com/CovidShield/server/pkg/config"
	pb "github.com/CovidShield/server/pkg/proto/covidshield"
	"github.com/CovidShield/server/pkg/timemath"
	"github.com/Shopify/goose/logger"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

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

func TestRetrieve(t *testing.T) {

	// Init config
	config.InitConfig()

	// Capture logs
	oldLog := log
	defer func() { log = oldLog }()

	nullLog, hook := test.NewNullLogger()
	nullLog.ExitFunc = func(code int) {}

	log = func(ctx logger.Valuer, err ...error) *logrus.Entry {
		return logrus.NewEntry(nullLog)
	}

	db := &persistence.Conn{}
	auth := &retrieval.Authenticator{}
	signer := &retrieval.Signer{}

	region := "302"
	goodAuth := "abcd"
	badAuth := "dcba"
	currentRSIN := pb.CurrentRollingStartIntervalNumber()
	currentDateNumber := fmt.Sprint(timemath.CurrentDateNumber())
	futureDate := fmt.Sprint(timemath.CurrentDateNumber() + 1)
	tooOldDate := fmt.Sprint(timemath.CurrentDateNumber() - 15)
	yesterdaysDate := fmt.Sprint(timemath.CurrentDateNumber() - 1)

	// Bad Auth Mock
	auth.On("Authenticate", region, currentDateNumber, badAuth).Return(false)

	// Good auth Mock
	auth.On("Authenticate", region, currentDateNumber, goodAuth).Return(true)

	// All keys download Mock
	auth.On("Authenticate", region, "00000", goodAuth).Return(true)
	startHour := (timemath.CurrentDateNumber() - 15) * 24
	endHour := timemath.CurrentDateNumber() * 24

	db.On("FetchKeysForHours", region, startHour, endHour, currentRSIN).Return([]*pb.TemporaryExposureKey{randomTestKey(), randomTestKey()}, nil)

	signer.On("Sign", mock.AnythingOfType("[]uint8")).Return(make([]byte, 64), nil)

	// Future date mock
	auth.On("Authenticate", region, futureDate, goodAuth).Return(true)

	// Too old date mock
	auth.On("Authenticate", region, tooOldDate, goodAuth).Return(true)

	// Failed DB Call
	auth.On("Authenticate", region, yesterdaysDate, goodAuth).Return(true)
	startHour = (timemath.CurrentDateNumber() - 1) * 24
	endHour = timemath.CurrentDateNumber() * 24

	db.On("FetchKeysForHours", region, startHour, endHour, currentRSIN).Return([]*pb.TemporaryExposureKey{}, fmt.Errorf("error"))

	servlet := NewRetrieveServlet(db, auth, signer)
	router := Router()
	servlet.RegisterRouting(router)

	// Bad authentication
	req, _ := http.NewRequest("GET", fmt.Sprintf("/retrieve/%s/%s/%s", region, currentDateNumber, badAuth), nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 401, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "invalid auth parameter", hook.LastEntry().Message)
	hook.Reset()

	hook.Reset()

	// Bad Method
	req, _ = http.NewRequest("POST", fmt.Sprintf("/retrieve/%s/%s/%s", region, currentDateNumber, goodAuth), nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 405, resp.Code, "Method not allowed response is expected")
	assert.Equal(t, "method not allowed\n", string(resp.Body.Bytes()), "Correct response is expected")

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "method not allowed", hook.LastEntry().Message)
	hook.Reset()

	// Get all keys for past 14 days
	req, _ = http.NewRequest("GET", fmt.Sprintf("/retrieve/%s/%s/%s", region, "00000", goodAuth), nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code, "Success response is expected")
	assert.Contains(t, resp.Header()["Content-Type"], "application/zip", "Cache-Control should be set to application/zip")
	assert.Contains(t, resp.Header()["Cache-Control"], "public, max-age=3600, max-stale=600", "Cache-Control should be set to public, max-age=3600, max-stale=600")

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.InfoLevel, hook.LastEntry().Level)
	assert.Equal(t, "Wrote retrieval", hook.LastEntry().Message)
	hook.Reset()

	// Get future keys
	req, _ = http.NewRequest("GET", fmt.Sprintf("/retrieve/%s/%s/%s", region, futureDate, goodAuth), nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 404, resp.Code, "404 response is expected")

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "request for future data", hook.LastEntry().Message)
	hook.Reset()

	// Get too old keys
	req, _ = http.NewRequest("GET", fmt.Sprintf("/retrieve/%s/%s/%s", region, tooOldDate, goodAuth), nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 410, resp.Code, "410 response is expected")

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "request for too-old data", hook.LastEntry().Message)
	hook.Reset()

	// Failing DB message
	req, _ = http.NewRequest("GET", fmt.Sprintf("/retrieve/%s/%s/%s", region, yesterdaysDate, goodAuth), nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 500, resp.Code, "500 response is expected")

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.ErrorLevel, hook.LastEntry().Level)
	assert.Equal(t, "database error", hook.LastEntry().Message)
	hook.Reset()

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
