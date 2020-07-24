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
	pb "github.com/CovidShield/server/pkg/proto/covidshield"
	"github.com/Shopify/goose/logger"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/crypto/nacl/box"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stretchr/testify/assert"
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
	goodAppPub, _, _ := box.GenerateKey(rand.Reader)

	db.On("PrivForPub", badServerPub[:]).Return(nil, fmt.Errorf("No priv cert"))
	db.On("PrivForPub", goodServerPub[:]).Return(goodServerPriv[:], nil)
	db.On("PrivForPub", goodServerPubBadPriv[:]).Return(make([]byte, 16), nil)

	servlet := NewUploadServlet(db)
	router := Router()
	servlet.RegisterRouting(router)

	// Bad, non-protobuff payload
	req, _ := http.NewRequest("POST", "/upload", strings.NewReader("sd"))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")

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
	encrypted = box.Seal(msg[:], marshalledUpload, &nonce, goodAppPub, goodServerPriv)

	payload, _ = proto.Marshal(buildUploadRequest(goodServerPub[:], nonce[:], goodAppPub[:], encrypted))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")

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
	encrypted = box.Seal(msg[:], marshalledUpload, &nonce, goodAppPub, goodServerPriv)

	payload, _ = proto.Marshal(buildUploadRequest(goodServerPub[:], nonce[:], goodAppPub[:], encrypted))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")

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
	encrypted = box.Seal(msg[:], marshalledUpload, &nonce, goodAppPub, goodServerPriv)

	payload, _ = proto.Marshal(buildUploadRequest(goodServerPub[:], nonce[:], goodAppPub[:], encrypted))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "invalid timestamp", hook.LastEntry().Message)
	hook.Reset()
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
