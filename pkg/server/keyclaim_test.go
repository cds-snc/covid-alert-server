package server

import (
	"bytes"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	keyclaim "github.com/cds-snc/covid-alert-server/mocks/pkg/keyclaim"
	persistence "github.com/cds-snc/covid-alert-server/mocks/pkg/persistence"
	"github.com/cds-snc/covid-alert-server/pkg/config"
	err "github.com/cds-snc/covid-alert-server/pkg/persistence"
	pb "github.com/cds-snc/covid-alert-server/pkg/proto/covidshield"
	"github.com/Shopify/goose/logger"
	"github.com/golang/protobuf/ptypes"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/crypto/nacl/box"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/stretchr/testify/assert"
)

func TestNewKeyClaimServlet(t *testing.T) {
	db := &persistence.Conn{}
	auth := &keyclaim.Authenticator{}

	expected := &keyClaimServlet{
		db:   db,
		auth: auth,
	}
	assert.Equal(t, expected, NewKeyClaimServlet(db, auth), "should return a new keyClaimServlet struct")
}

func TestRegisterRoutingKeyClaim(t *testing.T) {
	servlet := NewKeyClaimServlet(&persistence.Conn{}, &keyclaim.Authenticator{})
	router := Router()
	servlet.RegisterRouting(router)

	expectedPaths := GetPaths(router)
	assert.Contains(t, expectedPaths, "/new-key-claim", "should include a /new-key-claim path")
	assert.Contains(t, expectedPaths, "/new-key-claim/{hashID:[0-9,a-z]{128}}", "should include a /new-key-claim/{hashID:[0-9,a-z]{128}} path")
	assert.Contains(t, expectedPaths, "/claim-key", "should include a claim-key path")
}

func TestNewKeyClaim(t *testing.T) {
	db := &persistence.Conn{}
	auth := &keyclaim.Authenticator{}

	// Auth Mock
	auth.On("Authenticate", "badtoken").Return("", false)
	auth.On("Authenticate", "goodtoken").Return("302", true)
	auth.On("Authenticate", "errortoken").Return("302", true)

	auth.On("RegionFromAuthHeader", "").Return("302", "goodtoken",false)
	auth.On("RegionFromAuthHeader", "Bear thisisaverylongtoken").Return("", "", false)
	hashID := hex.EncodeToString(SHA512([]byte("abcd")))

	// Until we get a mockable time library we'll have to be less picky about events here
	db.On( "SaveEvent", mock.AnythingOfType("persistence.Event")).Return(nil)

	// DB Mock
	db.On("NewKeyClaim",mock.Anything, "302", "goodtoken", "").Return("AAABBBCCCC", nil)
	db.On("NewKeyClaim",mock.Anything, "302", "goodtoken", hashID).Return("AAABBBCCCC", nil)

	db.On("NewKeyClaim", mock.Anything, "302", "errortoken", "").Return("", fmt.Errorf("Random error"))
	db.On("NewKeyClaim", mock.Anything, "302", "errortoken", hashID).Return("", err.ErrHashIDClaimed)

	servlet := NewKeyClaimServlet(db, auth)
	router := Router()
	servlet.RegisterRouting(router)

	// Capture logs
	oldLog := log
	defer func() { log = oldLog }()

	nullLog, hook := test.NewNullLogger()
	nullLog.ExitFunc = func(code int) {}

	log = func(ctx logger.Valuer, err ...error) *logrus.Entry {
		return logrus.NewEntry(nullLog)
	}

	// Return CORS options header
	req, _ := http.NewRequest("OPTIONS", "/new-key-claim", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code, "OK response is expected")
	assert.Contains(t, resp.Header()["Access-Control-Allow-Origin"], config.AppConstants.CORSAccessControlAllowOrigin, "Access-Control-Allow-Origin should be set to the config value")
	assert.Contains(t, resp.Header()["Access-Control-Allow-Methods"], "POST", "Access-Control-Allow-Methods should be set to POST")
	assert.Contains(t, resp.Header()["Access-Control-Allow-Headers"], "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Referer, User-Agent", "Access-Control-Allow-Headers should be set to Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Referer, User-Agent")

	// No a POST request
	req, _ = http.NewRequest("GET", "/new-key-claim", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 401, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	assertLog(t, hook, 1, logrus.InfoLevel, "disallowed method")

	// No auth header
	req, _ = http.NewRequest("POST", "/new-key-claim", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 401, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	assertLog(t, hook, 1, logrus.InfoLevel, "bad auth header")

	// Malformed auth header - Bear
	req, _ = http.NewRequest("POST", "/new-key-claim", nil)
	req.Header.Set("Authorization", "Bear thisisaverylongtoken")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 401, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	assertLog(t, hook, 1, logrus.InfoLevel, "bad auth header")

	// Malformed auth header - No space
	req, _ = http.NewRequest("POST", "/new-key-claim", nil)
	req.Header.Set("Authorization", "Bearerthisisaverylongtoken")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 401, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	assertLog(t, hook, 1, logrus.InfoLevel, "bad auth header")

	// Bad auth token
	req, _ = http.NewRequest("POST", "/new-key-claim", nil)
	req.Header.Set("Authorization", "Bearer badtoken")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 401, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	assertLog(t, hook, 1, logrus.InfoLevel, "bad auth header")

	// Good auth token - no HashID
	req, _ = http.NewRequest("POST", "/new-key-claim", nil)
	req.Header.Set("Authorization", "Bearer goodtoken")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code, "Success response is expected")
	assert.Equal(t, "AAABBBCCCC\n", string(resp.Body.Bytes()), "Correct response is expected")

	// Good auth token -  HashID
	req, _ = http.NewRequest("POST", "/new-key-claim/"+hashID, nil)
	req.Header.Set("Authorization", "Bearer goodtoken")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code, "Success response is expected")
	assert.Equal(t, "AAABBBCCCC\n", string(resp.Body.Bytes()), "Correct response is expected")

	// Error saving - no HashID
	req, _ = http.NewRequest("POST", "/new-key-claim", nil)
	req.Header.Set("Authorization", "Bearer errortoken")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 500, resp.Code, "Server error response is expected")
	assert.Equal(t, "server error\n", string(resp.Body.Bytes()), "server error response is expected")

	assertLog(t, hook, 1, logrus.ErrorLevel, "error constructing new key claim")

	// Error saving - duplicate HashID
	req, _ = http.NewRequest("POST", "/new-key-claim/"+hashID, nil)
	req.Header.Set("Authorization", "Bearer errortoken")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 403, resp.Code, "forbidden response is expected")
	assert.Equal(t, "forbidden\n", string(resp.Body.Bytes()), "forbidden response is expected")

	assertLog(t, hook, 1, logrus.InfoLevel, "hashID used")
}

func TestClaimKey(t *testing.T) {
	db := &persistence.Conn{}
	auth := &keyclaim.Authenticator{}

	banDuration := (time.Duration(config.AppConstants.ClaimKeyBanDuration) * time.Hour)
	triesRemaining := config.AppConstants.MaxConsecutiveClaimKeyFailures

	// DB Mock
	db.On("CheckClaimKeyBan", "1.1.1.1").Return(0, time.Duration(0), fmt.Errorf("Random error"))
	db.On("CheckClaimKeyBan", "2.2.2.2").Return(0, banDuration, nil)
	db.On("CheckClaimKeyBan", "3.3.3.3").Return(triesRemaining, time.Duration(0), nil)
	db.On("CheckClaimKeyBan", "4.4.4.4").Return(triesRemaining, time.Duration(0), nil)
	db.On("CheckClaimKeyBan", "5.5.5.5").Return(triesRemaining, time.Duration(0), nil)

	appPub, _, _ := box.GenerateKey(rand.Reader)
	serverPub, _, _ := box.GenerateKey(rand.Reader)

	// Valid Code
	db.On("ClaimKey", "AAAAAAAAAA", appPub[:], mock.Anything).Return(serverPub[:], nil)

	// Error Code
	db.On("ClaimKey", "BBBBBBBBBB", appPub[:], mock.Anything).Return(nil, err.ErrInvalidKeyFormat)
	db.On("ClaimKey", "CCCCCCCCCC", appPub[:], mock.Anything).Return(nil, err.ErrDuplicateKey)
	db.On("ClaimKey", "DDDDDDDDDD", appPub[:], mock.Anything).Return(nil, err.ErrInvalidOneTimeCode)
	db.On("ClaimKey", "EEEEEEEEEE", appPub[:], mock.Anything).Return(nil, fmt.Errorf("Generic Error"))

	// Mock failure log
	db.On("ClaimKeyFailure", "3.3.3.3").Return(triesRemaining-1, banDuration, nil)
	db.On("ClaimKeyFailure", "4.4.4.4").Return(triesRemaining, time.Duration(0), fmt.Errorf("Random error"))

	//Clear IP failure
	db.On("ClaimKeySuccess", "3.3.3.3").Return(nil)
	db.On("ClaimKeySuccess", "5.5.5.5").Return(fmt.Errorf("Generic Error"))

	servlet := NewKeyClaimServlet(db, auth)
	router := Router()
	servlet.RegisterRouting(router)

	// Capture logs
	oldLog := log
	defer func() { log = oldLog }()

	nullLog, hook := test.NewNullLogger()
	nullLog.ExitFunc = func(code int) {}

	log = func(ctx logger.Valuer, err ...error) *logrus.Entry {
		return logrus.NewEntry(nullLog)
	}

	// Error finding keyclaim ban
	req, _ := http.NewRequest("POST", "/claim-key", nil)
	req.Header.Set("X-FORWARDED-FOR", "1.1.1.1")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 500, resp.Code, "Server error response is expected")
	assert.True(t, checkClaimKeyResponseError(resp.Body.Bytes(), pb.KeyClaimResponse_SERVER_ERROR))

	assertLog(t, hook, 1, logrus.ErrorLevel, "database error checking claim-key ban")

	// IP is banned
	req, _ = http.NewRequest("POST", "/claim-key", nil)
	req.Header.Set("X-FORWARDED-FOR", "2.2.2.2")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 429, resp.Code, "Too many requests response is expected")
	assert.True(t, checkClaimKeyResponseDuration(resp.Body.Bytes(), ptypes.DurationProto(banDuration)))

	assertLog(t, hook, 1, logrus.WarnLevel, "error reading request")

	// IP is banned - RemoteAddr
	req, _ = http.NewRequest("POST", "/claim-key", nil)
	req.RemoteAddr = "2.2.2.2"
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 429, resp.Code, "Too many requests response is expected")
	assert.True(t, checkClaimKeyResponseDuration(resp.Body.Bytes(), ptypes.DurationProto(banDuration)))

	assertLog(t, hook, 1, logrus.WarnLevel, "error reading request")

	// IP is banned - RemoteAddr with port
	req, _ = http.NewRequest("POST", "/claim-key", nil)
	req.RemoteAddr = "2.2.2.2:3333"
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 429, resp.Code, "Too many requests response is expected")
	assert.True(t, checkClaimKeyResponseDuration(resp.Body.Bytes(), ptypes.DurationProto(banDuration)))

	assertLog(t, hook, 1, logrus.WarnLevel, "error reading request")

	// Bad, non-protobuff payload
	req, _ = http.NewRequest("POST", "/claim-key", strings.NewReader("sd"))
	req.Header.Set("X-FORWARDED-FOR", "3.3.3.3")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "bad request response is expected")
	assert.True(t, checkClaimKeyResponseError(resp.Body.Bytes(), pb.KeyClaimResponse_UNKNOWN))
	assert.True(t, checkClaimKeyResponseTriesRemaining(resp.Body.Bytes(), uint32(triesRemaining)))

	assertLog(t, hook, 1, logrus.WarnLevel, "error unmarshalling request")

	// Invalid app key format
	code := "BBBBBBBBBB"
	upload := buildKeyClaimRequest(&code, appPub[:])
	marshalledUpload, _ := proto.Marshal(upload)

	req, _ = http.NewRequest("POST", "/claim-key", bytes.NewReader(marshalledUpload))
	req.Header.Set("X-FORWARDED-FOR", "3.3.3.3")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "bad request response is expected")
	assert.True(t, checkClaimKeyResponseError(resp.Body.Bytes(), pb.KeyClaimResponse_INVALID_KEY))
	assert.True(t, checkClaimKeyResponseTriesRemaining(resp.Body.Bytes(), uint32(triesRemaining)))

	assertLog(t, hook, 1, logrus.WarnLevel, "invalid key format")

	// Duplicate app key
	code = "CCCCCCCCCC"
	upload = buildKeyClaimRequest(&code, appPub[:])
	marshalledUpload, _ = proto.Marshal(upload)

	req, _ = http.NewRequest("POST", "/claim-key", bytes.NewReader(marshalledUpload))
	req.Header.Set("X-FORWARDED-FOR", "3.3.3.3")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 401, resp.Code, "unauthorised response is expected")
	assert.True(t, checkClaimKeyResponseError(resp.Body.Bytes(), pb.KeyClaimResponse_INVALID_KEY))
	assert.True(t, checkClaimKeyResponseTriesRemaining(resp.Body.Bytes(), uint32(triesRemaining)))

	assertLog(t, hook, 1, logrus.WarnLevel, "duplicate key")

	// Invalid one time code
	code = "DDDDDDDDDD"
	upload = buildKeyClaimRequest(&code, appPub[:])
	marshalledUpload, _ = proto.Marshal(upload)

	req, _ = http.NewRequest("POST", "/claim-key", bytes.NewReader(marshalledUpload))
	req.Header.Set("X-FORWARDED-FOR", "3.3.3.3")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 401, resp.Code, "unauthorised response is expected")
	assert.True(t, checkClaimKeyResponseError(resp.Body.Bytes(), pb.KeyClaimResponse_INVALID_ONE_TIME_CODE))
	assert.True(t, checkClaimKeyResponseTriesRemaining(resp.Body.Bytes(), uint32(triesRemaining)-1))
	assert.True(t, checkClaimKeyResponseDuration(resp.Body.Bytes(), ptypes.DurationProto(banDuration)))

	assertLog(t, hook, 1, logrus.WarnLevel, "invalid one time code")

	// Invalid one time code - DB failure on IP ban check
	code = "DDDDDDDDDD"
	upload = buildKeyClaimRequest(&code, appPub[:])
	marshalledUpload, _ = proto.Marshal(upload)

	req, _ = http.NewRequest("POST", "/claim-key", bytes.NewReader(marshalledUpload))
	req.Header.Set("X-FORWARDED-FOR", "4.4.4.4")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 500, resp.Code, "internal server error response is expected")
	assert.True(t, checkClaimKeyResponseError(resp.Body.Bytes(), pb.KeyClaimResponse_SERVER_ERROR))
	assert.True(t, checkClaimKeyResponseTriesRemaining(resp.Body.Bytes(), uint32(triesRemaining)))

	assertLog(t, hook, 1, logrus.ErrorLevel, "database error recording claim-key failure")

	// Generic error
	code = "EEEEEEEEEE"
	upload = buildKeyClaimRequest(&code, appPub[:])
	marshalledUpload, _ = proto.Marshal(upload)

	req, _ = http.NewRequest("POST", "/claim-key", bytes.NewReader(marshalledUpload))
	req.Header.Set("X-FORWARDED-FOR", "3.3.3.3")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 500, resp.Code, "internal server error response is expected")
	assert.True(t, checkClaimKeyResponseError(resp.Body.Bytes(), pb.KeyClaimResponse_SERVER_ERROR))
	assert.True(t, checkClaimKeyResponseTriesRemaining(resp.Body.Bytes(), uint32(triesRemaining)))

	assertLog(t, hook, 1, logrus.ErrorLevel, "failure to claim key using OneTimeCode")

	// Success with normal code
	code = "AAAAAAAAAA"
	upload = buildKeyClaimRequest(&code, appPub[:])
	marshalledUpload, _ = proto.Marshal(upload)

	req, _ = http.NewRequest("POST", "/claim-key", bytes.NewReader(marshalledUpload))
	req.Header.Set("X-FORWARDED-FOR", "3.3.3.3")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code, "success response is expected")
	assert.True(t, checkClaimKeyResponseError(resp.Body.Bytes(), pb.KeyClaimResponse_NONE))
	assert.True(t, checkClaimKeyResponseTriesRemaining(resp.Body.Bytes(), uint32(triesRemaining)))

	// Success with hyphenated code
	code = "AAA-AAA-AAAA"
	upload = buildKeyClaimRequest(&code, appPub[:])
	marshalledUpload, _ = proto.Marshal(upload)

	req, _ = http.NewRequest("POST", "/claim-key", bytes.NewReader(marshalledUpload))
	req.Header.Set("X-FORWARDED-FOR", "3.3.3.3")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code, "success response is expected")
	assert.True(t, checkClaimKeyResponseError(resp.Body.Bytes(), pb.KeyClaimResponse_NONE))
	assert.True(t, checkClaimKeyResponseTriesRemaining(resp.Body.Bytes(), uint32(triesRemaining)))

	// Success with spaced code
	code = "AAA AAA AAAA  "
	upload = buildKeyClaimRequest(&code, appPub[:])
	marshalledUpload, _ = proto.Marshal(upload)

	req, _ = http.NewRequest("POST", "/claim-key", bytes.NewReader(marshalledUpload))
	req.Header.Set("X-FORWARDED-FOR", "3.3.3.3")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code, "success response is expected")
	assert.True(t, checkClaimKeyResponseError(resp.Body.Bytes(), pb.KeyClaimResponse_NONE))
	assert.True(t, checkClaimKeyResponseTriesRemaining(resp.Body.Bytes(), uint32(triesRemaining)))

	// Success with normal code - but error clearing IP
	code = "AAAAAAAAAA"
	upload = buildKeyClaimRequest(&code, appPub[:])
	marshalledUpload, _ = proto.Marshal(upload)

	req, _ = http.NewRequest("POST", "/claim-key", bytes.NewReader(marshalledUpload))
	req.Header.Set("X-FORWARDED-FOR", "5.5.5.5")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code, "success response is expected")
	assert.True(t, checkClaimKeyResponseError(resp.Body.Bytes(), pb.KeyClaimResponse_NONE))
	assert.True(t, checkClaimKeyResponseTriesRemaining(resp.Body.Bytes(), uint32(triesRemaining)))

	assertLog(t, hook, 1, logrus.WarnLevel, "error recording claim-key success")
}

func checkClaimKeyResponseDuration(data []byte, duration *durationpb.Duration) bool {
	var response pb.KeyClaimResponse
	proto.Unmarshal(data, &response)
	return response.GetRemainingBanDuration().Seconds == duration.Seconds
}

func checkClaimKeyResponseError(data []byte, expectedCode pb.KeyClaimResponse_ErrorCode) bool {
	var response pb.KeyClaimResponse
	proto.Unmarshal(data, &response)
	return response.GetError() == expectedCode
}

func checkClaimKeyResponseTriesRemaining(data []byte, triesRemaining uint32) bool {
	var response pb.KeyClaimResponse
	proto.Unmarshal(data, &response)
	return response.GetTriesRemaining() == triesRemaining
}

func buildKeyClaimRequest(oneTimeCode *string, appPublicKey []byte) *pb.KeyClaimRequest {
	return &pb.KeyClaimRequest{
		OneTimeCode:  oneTimeCode,
		AppPublicKey: appPublicKey,
	}
}

func assertLog(t *testing.T, hook *test.Hook, length int, level logrus.Level, msg string) {
	assert.Equal(t, length, len(hook.Entries))
	assert.Equal(t, level, hook.LastEntry().Level)
	assert.Equal(t, msg, hook.LastEntry().Message)
	hook.Reset()
}

func SHA512(message []byte) []byte {
	c := sha512.New()
	c.Write(message)
	return c.Sum(nil)
}
