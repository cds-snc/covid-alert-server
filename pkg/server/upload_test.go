package server

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	persistence "github.com/CovidShield/server/mocks/pkg/persistence"
	persistenceErrors "github.com/CovidShield/server/pkg/persistence"
	pb "github.com/CovidShield/server/pkg/proto/covidshield"
	"github.com/Shopify/goose/logger"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/crypto/nacl/box"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewUploadServlet(t *testing.T) {
	db := &persistence.Conn{}

	expected := &uploadServlet{
		db: db,
	}
	assert.Equal(t, expected, NewUploadServlet(db), "should return a new uploadServlet struct")
}

func TestRegisterRoutingUpload(t *testing.T) {
	servlet := NewUploadServlet(&persistence.Conn{})
	router := Router()
	servlet.RegisterRouting(router)

	expectedPaths := GetPaths(router)
	assert.Contains(t, expectedPaths, "/upload", "should include an upload path")
}

func TestUploadError(t *testing.T) {
	err := pb.EncryptedUploadResponse_UNKNOWN
	expected := &pb.EncryptedUploadResponse{Error: &err}
	assert.Equal(t, expected, uploadError(err), "should wrap the upload error code in an upload error response")
}

func TestUpload(t *testing.T) {
	// Capture logs
	oldLog := log
	defer func() { log = oldLog }()

	nullLog, hook := test.NewNullLogger()
	nullLog.ExitFunc = func(code int) {}

	log = func(ctx logger.Valuer, err ...error) *logrus.Entry {
		return logrus.NewEntry(nullLog)
	}

	db := &persistence.Conn{}

	// Set up PrivForPub
	badServerPub, _, _ := box.GenerateKey(rand.Reader)
	goodServerPub, goodServerPriv, _ := box.GenerateKey(rand.Reader)
	goodServerPubBadPriv, _, _ := box.GenerateKey(rand.Reader)
	goodAppPub, goodAppPriv, _ := box.GenerateKey(rand.Reader)
	goodAppPubKeyUsed, goodAppPrivKeyUsed, _ := box.GenerateKey(rand.Reader)
	goodAppPubNoKeysRemaining, goodAppPrivNoKeysRemaining, _ := box.GenerateKey(rand.Reader)
	goodServerPubNoKeysRemaining, goodServerPrivNoKeysRemaining, _ := box.GenerateKey(rand.Reader)
	goodAppPubDBError, goodAppPrivDBError, _ := box.GenerateKey(rand.Reader)

	db.On("PrivForPub", badServerPub[:]).Return(nil, fmt.Errorf("No priv cert"))
	db.On("PrivForPub", goodServerPub[:]).Return(goodServerPriv[:], nil)
	db.On("PrivForPub", goodServerPubNoKeysRemaining[:]).Return(goodServerPrivNoKeysRemaining[:], nil)
	db.On("PrivForPub", goodServerPubBadPriv[:]).Return(make([]byte, 16), nil)

	db.On("StoreKeys", goodAppPubKeyUsed, mock.AnythingOfType("[]*covidshield.TemporaryExposureKey")).Return(persistenceErrors.ErrKeyConsumed)
	db.On("StoreKeys", goodAppPubNoKeysRemaining, mock.AnythingOfType("[]*covidshield.TemporaryExposureKey")).Return(persistenceErrors.ErrTooManyKeys)
	db.On("StoreKeys", goodAppPubDBError, mock.AnythingOfType("[]*covidshield.TemporaryExposureKey")).Return(fmt.Errorf("generic DB error"))
	db.On("StoreKeys", goodAppPub, mock.AnythingOfType("[]*covidshield.TemporaryExposureKey")).Return(nil)

	servlet := NewUploadServlet(db)
	router := Router()
	servlet.RegisterRouting(router)

	// Bad, non-protobuff payload
	req, _ := http.NewRequest("POST", "/upload", strings.NewReader("sd"))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_UNKNOWN))

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "error unmarshalling request", hook.LastEntry().Message)
	hook.Reset()

	// Server Public cert too short
	payload, _ := proto.Marshal(buildUploadRequest(make([]byte, 16), nil, nil, nil))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_INVALID_CRYPTO_PARAMETERS))

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "server public key was not expected length", hook.LastEntry().Message)
	hook.Reset()

	// Public cert not found
	payload, _ = proto.Marshal(buildUploadRequest(badServerPub[:], nil, nil, nil))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 401, resp.Code, "401 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_INVALID_KEYPAIR))

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "failure to resolve client keypair", hook.LastEntry().Message)
	hook.Reset()

	// Nonce incorrect length
	payload, _ = proto.Marshal(buildUploadRequest(goodServerPub[:], make([]byte, 16), nil, nil))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_INVALID_CRYPTO_PARAMETERS))

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "nonce was not expected length", hook.LastEntry().Message)
	hook.Reset()

	// App Public cert too short
	payload, _ = proto.Marshal(buildUploadRequest(goodServerPub[:], make([]byte, 24), make([]byte, 16), nil))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_INVALID_CRYPTO_PARAMETERS))

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "app public key key was not expected length", hook.LastEntry().Message)
	hook.Reset()

	// Server private cert too short
	payload, _ = proto.Marshal(buildUploadRequest(goodServerPubBadPriv[:], make([]byte, 24), make([]byte, 32), nil))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 500, resp.Code, "500 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_SERVER_ERROR))

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.ErrorLevel, hook.LastEntry().Level)
	assert.Equal(t, "server private key was not expected length", hook.LastEntry().Message)
	hook.Reset()

	// Fails to decrypt payload
	var nonce [24]byte
	var msg []byte
	io.ReadFull(rand.Reader, nonce[:])
	encrypted := box.Seal(msg[:], []byte("hello world"), &nonce, goodAppPub, badServerPub)

	payload, _ = proto.Marshal(buildUploadRequest(goodServerPub[:], nonce[:], goodAppPub[:], encrypted))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_DECRYPTION_FAILED))

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "failure to decrypt payload", hook.LastEntry().Message)
	hook.Reset()

	// Fails unmarshall into Upload
	io.ReadFull(rand.Reader, nonce[:])
	encrypted = box.Seal(msg[:], []byte("hello world"), &nonce, goodAppPub, goodServerPriv)

	payload, _ = proto.Marshal(buildUploadRequest(goodServerPub[:], nonce[:], goodAppPub[:], encrypted))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_INVALID_PAYLOAD))

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "error unmarshalling request payload", hook.LastEntry().Message)
	hook.Reset()

	// No keys in payload
	io.ReadFull(rand.Reader, nonce[:])
	ts := time.Now()
	pbts := timestamppb.Timestamp{
		Seconds: ts.Unix(),
	}
	upload := buildUpload(0, pbts)
	marshalledUpload, _ := proto.Marshal(upload)
	encrypted = box.Seal(msg[:], marshalledUpload, &nonce, goodServerPub, goodAppPriv)

	payload, _ = proto.Marshal(buildUploadRequest(goodServerPub[:], nonce[:], goodAppPub[:], encrypted))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_NO_KEYS_IN_PAYLOAD))

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "no keys provided", hook.LastEntry().Message)
	hook.Reset()

	// Too many keys in payload
	io.ReadFull(rand.Reader, nonce[:])
	ts = time.Now()
	pbts = timestamppb.Timestamp{
		Seconds: ts.Unix(),
	}
	upload = buildUpload(pb.MaxKeysInUpload+1, pbts)
	marshalledUpload, _ = proto.Marshal(upload)
	encrypted = box.Seal(msg[:], marshalledUpload, &nonce, goodServerPub, goodAppPriv)

	payload, _ = proto.Marshal(buildUploadRequest(goodServerPub[:], nonce[:], goodAppPub[:], encrypted))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_TOO_MANY_KEYS))

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "too many keys provided", hook.LastEntry().Message)
	hook.Reset()

	// Invalid timestamp
	io.ReadFull(rand.Reader, nonce[:])
	ts = time.Now()
	pbts = timestamppb.Timestamp{
		Seconds: ts.Unix() - 4000,
	}
	upload = buildUpload(pb.MaxKeysInUpload, pbts)
	marshalledUpload, _ = proto.Marshal(upload)
	encrypted = box.Seal(msg[:], marshalledUpload, &nonce, goodServerPub, goodAppPriv)

	payload, _ = proto.Marshal(buildUploadRequest(goodServerPub[:], nonce[:], goodAppPub[:], encrypted))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_INVALID_TIMESTAMP))

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "invalid timestamp", hook.LastEntry().Message)
	hook.Reset()

	// Expired Key
	io.ReadFull(rand.Reader, nonce[:])
	ts = time.Now()
	pbts = timestamppb.Timestamp{
		Seconds: ts.Unix(),
	}
	upload = buildUpload(1, pbts)
	marshalledUpload, _ = proto.Marshal(upload)
	encrypted = box.Seal(msg[:], marshalledUpload, &nonce, goodServerPub, goodAppPrivKeyUsed)

	payload, _ = proto.Marshal(buildUploadRequest(goodServerPub[:], nonce[:], goodAppPubKeyUsed[:], encrypted))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_INVALID_KEYPAIR))

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "key is used up", hook.LastEntry().Message)
	hook.Reset()

	// Not enough keys remaining
	io.ReadFull(rand.Reader, nonce[:])
	ts = time.Now()
	pbts = timestamppb.Timestamp{
		Seconds: ts.Unix(),
	}
	upload = buildUpload(1, pbts)
	marshalledUpload, _ = proto.Marshal(upload)
	encrypted = box.Seal(msg[:], marshalledUpload, &nonce, goodServerPubNoKeysRemaining, goodAppPrivNoKeysRemaining)

	payload, _ = proto.Marshal(buildUploadRequest(goodServerPubNoKeysRemaining[:], nonce[:], goodAppPubNoKeysRemaining[:], encrypted))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_TOO_MANY_KEYS))

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "not enough keys remaining", hook.LastEntry().Message)
	hook.Reset()

	// Generic DB Error
	io.ReadFull(rand.Reader, nonce[:])
	ts = time.Now()
	pbts = timestamppb.Timestamp{
		Seconds: ts.Unix(),
	}
	upload = buildUpload(1, pbts)
	marshalledUpload, _ = proto.Marshal(upload)
	encrypted = box.Seal(msg[:], marshalledUpload, &nonce, goodServerPub, goodAppPrivDBError)

	payload, _ = proto.Marshal(buildUploadRequest(goodServerPub[:], nonce[:], goodAppPubDBError[:], encrypted))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 500, resp.Code, "500 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_SERVER_ERROR))

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.ErrorLevel, hook.LastEntry().Level)
	assert.Equal(t, "failed to store diagnosis keys", hook.LastEntry().Message)
	hook.Reset()

	// Good reponse
	io.ReadFull(rand.Reader, nonce[:])
	ts = time.Now()
	pbts = timestamppb.Timestamp{
		Seconds: ts.Unix(),
	}
	upload = buildUpload(1, pbts)
	marshalledUpload, _ = proto.Marshal(upload)
	encrypted = box.Seal(msg[:], marshalledUpload, &nonce, goodServerPub, goodAppPriv)

	payload, _ = proto.Marshal(buildUploadRequest(goodServerPub[:], nonce[:], goodAppPub[:], encrypted))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code, "200 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_NONE))
}

func TestValidateKey(t *testing.T) {
	// Capture logs
	oldLog := log
	defer func() { log = oldLog }()

	nullLog, hook := test.NewNullLogger()
	nullLog.ExitFunc = func(code int) {}

	log = func(ctx logger.Valuer, err ...error) *logrus.Entry {
		return logrus.NewEntry(nullLog)
	}

	db := &persistence.Conn{}
	servlet := NewUploadServlet(db)
	router := Router()
	servlet.RegisterRouting(router)

	req, _ := http.NewRequest("POST", "/upload", nil)
	resp := httptest.NewRecorder()

	// Test RollingPeriod < 1
	token := make([]byte, 16)
	rand.Read(token)
	key := buildKey(token, int32(2), int32(2651450), int32(0))

	result := validateKey(req.Context(), resp, &key)

	assert.False(t, result)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_INVALID_ROLLING_PERIOD))

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "missing or invalid rollingPeriod", hook.LastEntry().Message)
	hook.Reset()

	// Test RollingPeriod > 144
	token = make([]byte, 16)
	rand.Read(token)
	key = buildKey(token, int32(2), int32(2651450), int32(145))

	result = validateKey(req.Context(), resp, &key)

	assert.False(t, result)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_INVALID_ROLLING_PERIOD))

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "missing or invalid rollingPeriod", hook.LastEntry().Message)
	hook.Reset()

	// Key data not 16 bytes
	token = make([]byte, 8)
	rand.Read(token)
	key = buildKey(token, int32(2), int32(2651450), int32(144))

	result = validateKey(req.Context(), resp, &key)

	assert.False(t, result)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_INVALID_KEY_DATA))

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "invalid key data", hook.LastEntry().Message)
	hook.Reset()

	// Invalid RSIN
	token = make([]byte, 16)
	rand.Read(token)
	key = buildKey(token, int32(2), int32(0), int32(144))

	result = validateKey(req.Context(), resp, &key)

	assert.False(t, result)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_INVALID_ROLLING_START_INTERVAL_NUMBER))

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "invalid rolling start number", hook.LastEntry().Message)
	hook.Reset()

	//  TransmissionRiskLevel < 0
	token = make([]byte, 16)
	rand.Read(token)
	key = buildKey(token, int32(-1), int32(2651450), int32(144))

	result = validateKey(req.Context(), resp, &key)

	assert.False(t, result)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_INVALID_TRANSMISSION_RISK_LEVEL))

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "invalid transmission risk level", hook.LastEntry().Message)
	hook.Reset()

	// Invalid TransmissionRiskLevel > 8
	token = make([]byte, 16)
	rand.Read(token)
	key = buildKey(token, int32(9), int32(2651450), int32(144))

	result = validateKey(req.Context(), resp, &key)

	assert.False(t, result)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_INVALID_TRANSMISSION_RISK_LEVEL))

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "invalid transmission risk level", hook.LastEntry().Message)
	hook.Reset()

	// Valid key
	token = make([]byte, 16)
	rand.Read(token)
	key = buildKey(token, int32(8), int32(2651450), int32(144))

	result = validateKey(req.Context(), resp, &key)

	assert.True(t, result)
}

func TestValidateKeys(t *testing.T) {
	// Capture logs
	oldLog := log
	defer func() { log = oldLog }()

	nullLog, hook := test.NewNullLogger()
	nullLog.ExitFunc = func(code int) {}

	log = func(ctx logger.Valuer, err ...error) *logrus.Entry {
		return logrus.NewEntry(nullLog)
	}

	db := &persistence.Conn{}
	servlet := NewUploadServlet(db)
	router := Router()
	servlet.RegisterRouting(router)

	req, _ := http.NewRequest("POST", "/upload", nil)
	resp := httptest.NewRecorder()

	// Returns false on bad key
	token := make([]byte, 16)
	rand.Read(token)
	key := buildKey(token, int32(2), int32(2651450), int32(0))

	result := validateKeys(req.Context(), resp, []*pb.TemporaryExposureKey{&key})
	assert.False(t, result)

	// Retuns false on keys where rsin is more than 15 days apart
	keyOne := buildKey(token, int32(2), int32(2651450), int32(144))
	keyTwo := buildKey(token, int32(2), int32(2651450-(144*15)), int32(144))

	result = validateKeys(req.Context(), resp, []*pb.TemporaryExposureKey{&keyOne, &keyTwo})
	assert.False(t, result)

	assert.Equal(t, 400, resp.Code, "400 response is expected")
	assert.True(t, checkUploadResponse(resp.Body.Bytes(), pb.EncryptedUploadResponse_INVALID_ROLLING_START_INTERVAL_NUMBER))

	assert.Equal(t, 2, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "sequence of rollingStartIntervalNumbers exceeds 15 days", hook.LastEntry().Message)
	hook.Reset()

	// Retuns true on good key
	key = buildKey(token, int32(2), int32(2651450), int32(144))

	result = validateKeys(req.Context(), resp, []*pb.TemporaryExposureKey{&key})
	assert.True(t, result)

}

func buildKey(token []byte, transmissionRiskLevel, rollingStartIntervalNumber, rollingPeriod int32) pb.TemporaryExposureKey {
	return pb.TemporaryExposureKey{
		KeyData:                    token,
		TransmissionRiskLevel:      &transmissionRiskLevel,
		RollingStartIntervalNumber: &rollingStartIntervalNumber,
		RollingPeriod:              &rollingPeriod,
	}
}

func buildUploadRequest(serverPubKey []byte, nonce []byte, appPublicKey []byte, payload []byte) *pb.EncryptedUploadRequest {
	upload := &pb.EncryptedUploadRequest{
		ServerPublicKey: serverPubKey,
		AppPublicKey:    appPublicKey,
		Nonce:           nonce,
		Payload:         payload,
	}
	return upload
}

func buildUpload(count int, ts timestamppb.Timestamp) *pb.Upload {
	var keys []*pb.TemporaryExposureKey
	for i := 0; i < count; i++ {
		keys = append(keys, randomTestKey())
	}
	upload := &pb.Upload{
		Keys:      keys,
		Timestamp: &ts,
	}
	return upload
}

func checkUploadResponse(data []byte, expectedCode pb.EncryptedUploadResponse_ErrorCode) bool {
	var response pb.EncryptedUploadResponse
	proto.Unmarshal(data, &response)
	return response.GetError() == expectedCode
}
