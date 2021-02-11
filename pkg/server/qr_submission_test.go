package server

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Shopify/goose/logger"
	"github.com/Shopify/goose/srvutil"
	keyclaim "github.com/cds-snc/covid-alert-server/mocks/pkg/keyclaim"
	persistence "github.com/cds-snc/covid-alert-server/mocks/pkg/persistence"
	pb "github.com/cds-snc/covid-alert-server/pkg/proto/covidshield"
	"github.com/cds-snc/covid-alert-server/pkg/testhelpers"
	timestamp "github.com/golang/protobuf/ptypes"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/proto"
)

func setupQrUploadRouter(db *persistence.Conn, auth *keyclaim.Authenticator) *mux.Router {
	servlet := NewQRSubmissionServlet(db, auth)
	router := Router()
	servlet.RegisterRouting(router)
	return router
}

func setupQrUploadTest() (*test.Hook, *logger.Logger, *persistence.Conn, *mux.Router) {

	hook, oldLog := testhelpers.SetupTestLogging(&log)
	db := &persistence.Conn{}
	auth := &keyclaim.Authenticator{}

	auth.On("Authenticate", "goodtoken").Return("302", true)
	auth.On("RegionFromAuthHeader", "Bearer goodtoken").Return("302", "goodtoken", true)

	router := setupQrUploadRouter(db, auth)

	return hook, oldLog, db, router

}

func TestNewQRSubmissionServlet(t *testing.T) {
	db := &persistence.Conn{}
	auth := &keyclaim.Authenticator{}

	expected := &qrSubmissionServlet{
		db:   db,
		auth: auth,
	}

	funcName1 := runtime.FuncForPC(reflect.ValueOf(srvutil.PrefixServlet(expected, "/services")).Pointer()).Name()
	funcName2 := runtime.FuncForPC(reflect.ValueOf(NewQRSubmissionServlet(db, auth)).Pointer()).Name()
	assert.Equal(t, funcName1, funcName2, "should return a new qrSubmissionServlet function")
}

func TestRegisterRouting(t *testing.T) {
	servlet := NewQRSubmissionServlet(&persistence.Conn{}, &keyclaim.Authenticator{})
	router := Router()
	servlet.RegisterRouting(router)

	expectedPaths := GetPaths(router)
	assert.Contains(t, expectedPaths, "/qr/new-submission", "should include a /qr/new-submission path")
}

func TestQrUploadResponse(t *testing.T) {
	err := pb.QrSubmissionResponse_UNKNOWN
	expected := &pb.QrSubmissionResponse{Error: &err}
	assert.Equal(t, expected, qrUploadResponse(err), "should wrap the qr upload error code in a qr upload error response")
}

func TestQrUpload_NoPost(t *testing.T) {
	hook, oldLog, _, router := setupQrUploadTest()
	defer func() { log = *oldLog }()

	req, _ := http.NewRequest("GET", "/qr/new-submission", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 401, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	testhelpers.AssertLog(t, hook, 1, logrus.InfoLevel, "disallowed method")
}

func TestQrUpload_BadAuthToken(t *testing.T) {

	auth := &keyclaim.Authenticator{}
	// Auth Mock
	auth.On("Authenticate", "badtoken").Return("", false)
	auth.On("RegionFromAuthHeader", "Bearer badtoken").Return("", "", false)

	db := &persistence.Conn{}
	router := setupQrUploadRouter(db, auth)
	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	// Bad auth token
	req, _ := http.NewRequest("POST", "/qr/new-submission", nil)
	req.Header.Set("Authorization", "Bearer badtoken")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 401, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	testhelpers.AssertLog(t, hook, 1, logrus.InfoLevel, "bad auth header")
}

func TestQrUpload_NonProtoBufPayload(t *testing.T) {
	hook, oldLog, _, router := setupQrUploadTest()
	defer func() { log = *oldLog }()

	// Bad, non-protobuff payload
	req, _ := http.NewRequest("POST", "/qr/new-submission", strings.NewReader("sd"))
	req.Header.Set("Authorization", "Bearer goodtoken")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkQrUploadResponse(resp.Body.Bytes(), pb.QrSubmissionResponse_UNKNOWN))

	testhelpers.AssertLog(t, hook, 1, logrus.WarnLevel, "error unmarshalling request")
}

func TestQrUpload_LocationIdTooShort(t *testing.T) {
	hook, oldLog, _, router := setupQrUploadTest()
	defer func() { log = *oldLog }()

	// Location ID too short
	uuid := "abcd"
	startTime, _ := timestamp.TimestampProto(time.Now())
	endTime, _ := timestamp.TimestampProto(time.Now().Add(time.Hour * 24))
	submission := pb.QrSubmission{LocationId: &uuid, StartTime: startTime, EndTime: endTime}

	payload, _ := proto.Marshal(&submission)
	req, _ := http.NewRequest("POST", "/qr/new-submission", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer goodtoken")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkQrUploadResponse(resp.Body.Bytes(), pb.QrSubmissionResponse_INVALID_ID))

	testhelpers.AssertLog(t, hook, 1, logrus.WarnLevel, "Location ID is not valid")
}

func TestQrUpload_LocationIdTooLong(t *testing.T) {
	hook, oldLog, _, router := setupQrUploadTest()
	defer func() { log = *oldLog }()

	// Location ID too short
	uuid := "abcdef-abcdef-abcdef-abcdef-abcdef-abcdef"
	startTime, _ := timestamp.TimestampProto(time.Now())
	endTime, _ := timestamp.TimestampProto(time.Now().Add(time.Hour * 24))
	submission := pb.QrSubmission{LocationId: &uuid, StartTime: startTime, EndTime: endTime}

	payload, _ := proto.Marshal(&submission)
	req, _ := http.NewRequest("POST", "/qr/new-submission", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer goodtoken")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkQrUploadResponse(resp.Body.Bytes(), pb.QrSubmissionResponse_INVALID_ID))

	testhelpers.AssertLog(t, hook, 1, logrus.WarnLevel, "Location ID is not valid")
}

func TestQrUpload_StartTimeZero(t *testing.T) {
	hook, oldLog, _, router := setupQrUploadTest()
	defer func() { log = *oldLog }()

	uuid := "8a2c34b2-74a5-4b6a-8bed-79b7823b37c7"
	startTime, _ := timestamp.TimestampProto(time.Unix(0, 0))
	endTime, _ := timestamp.TimestampProto(time.Now().Add(time.Hour * 24))
	submission := pb.QrSubmission{LocationId: &uuid, StartTime: startTime, EndTime: endTime}

	payload, _ := proto.Marshal(&submission)
	req, _ := http.NewRequest("POST", "/qr/new-submission", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer goodtoken")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkQrUploadResponse(resp.Body.Bytes(), pb.QrSubmissionResponse_MISSING_TIMESTAMP))

	testhelpers.AssertLog(t, hook, 1, logrus.WarnLevel, "missing/invalid timestamp")
}

func TestQrUpload_EndTimeZero(t *testing.T) {
	hook, oldLog, _, router := setupQrUploadTest()
	defer func() { log = *oldLog }()

	uuid := "8a2c34b2-74a5-4b6a-8bed-79b7823b37c7"
	startTime, _ := timestamp.TimestampProto(time.Now())
	endTime, _ := timestamp.TimestampProto(time.Unix(0, 0))
	submission := pb.QrSubmission{LocationId: &uuid, StartTime: startTime, EndTime: endTime}

	payload, _ := proto.Marshal(&submission)
	req, _ := http.NewRequest("POST", "/qr/new-submission", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer goodtoken")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkQrUploadResponse(resp.Body.Bytes(), pb.QrSubmissionResponse_MISSING_TIMESTAMP))

	testhelpers.AssertLog(t, hook, 1, logrus.WarnLevel, "missing/invalid timestamp")
}

func TestQrUpload_EndBeforeStart(t *testing.T) {
	hook, oldLog, _, router := setupQrUploadTest()
	defer func() { log = *oldLog }()

	uuid := "8a2c34b2-74a5-4b6a-8bed-79b7823b37c7"
	endTime, _ := timestamp.TimestampProto(time.Now())
	startTime, _ := timestamp.TimestampProto(time.Now().Add(time.Hour * 24))
	submission := pb.QrSubmission{LocationId: &uuid, StartTime: startTime, EndTime: endTime}

	payload, _ := proto.Marshal(&submission)
	req, _ := http.NewRequest("POST", "/qr/new-submission", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer goodtoken")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkQrUploadResponse(resp.Body.Bytes(), pb.QrSubmissionResponse_PERIOD_INVALID))

	testhelpers.AssertLog(t, hook, 1, logrus.WarnLevel, "invalid timeperiod")
}

func TestQrUpload_ErrorSaving(t *testing.T) {
	hook, oldLog, db, router := setupQrUploadTest()
	defer func() { log = *oldLog }()

	db.On("NewQrSubmission", mock.Anything, "goodtoken", mock.AnythingOfType("*covidshield.QrSubmission")).Return(fmt.Errorf("error"))

	uuid := "8a2c34b2-74a5-4b6a-8bed-79b7823b37c7"
	startTime, _ := timestamp.TimestampProto(time.Now())
	endTime, _ := timestamp.TimestampProto(time.Now().Add(time.Hour * 24))
	submission := pb.QrSubmission{LocationId: &uuid, StartTime: startTime, EndTime: endTime}

	payload, _ := proto.Marshal(&submission)
	req, _ := http.NewRequest("POST", "/qr/new-submission", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer goodtoken")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkQrUploadResponse(resp.Body.Bytes(), pb.QrSubmissionResponse_SERVER_ERROR))

	testhelpers.AssertLog(t, hook, 1, logrus.WarnLevel, "error saving QR submission")
}

func TestQrUpload_SuccessSaving(t *testing.T) {
	_, oldLog, db, router := setupQrUploadTest()
	defer func() { log = *oldLog }()

	db.On("NewQrSubmission", mock.Anything, "goodtoken", mock.AnythingOfType("*covidshield.QrSubmission")).Return(nil)

	uuid := "8a2c34b2-74a5-4b6a-8bed-79b7823b37c7"
	startTime, _ := timestamp.TimestampProto(time.Now())
	endTime, _ := timestamp.TimestampProto(time.Now().Add(time.Hour * 24))
	submission := pb.QrSubmission{LocationId: &uuid, StartTime: startTime, EndTime: endTime}

	payload, _ := proto.Marshal(&submission)
	req, _ := http.NewRequest("POST", "/qr/new-submission", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer goodtoken")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code, "200 response is expected")
	assert.True(t, checkQrUploadResponse(resp.Body.Bytes(), pb.QrSubmissionResponse_NONE))
}

func checkQrUploadResponse(data []byte, expectedCode pb.QrSubmissionResponse_ErrorCode) bool {
	var response pb.QrSubmissionResponse
	proto.Unmarshal(data, &response)
	return response.GetError() == expectedCode
}
