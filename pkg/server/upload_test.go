package server

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	persistence "github.com/CovidShield/server/mocks/pkg/persistence"
	pb "github.com/CovidShield/server/pkg/proto/covidshield"
	"github.com/Shopify/goose/logger"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/crypto/nacl/box"
	"google.golang.org/protobuf/proto"

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
	badPub, _, _ := box.GenerateKey(rand.Reader)
	goodPub, goodPriv, _ := box.GenerateKey(rand.Reader)
	goodPubBadPriv, _, _ := box.GenerateKey(rand.Reader)

	db.On("PrivForPub", badPub[:]).Return(nil, fmt.Errorf("No priv cert"))
	db.On("PrivForPub", goodPub[:]).Return(goodPriv[:], nil)
	db.On("PrivForPub", goodPubBadPriv[:]).Return(make([]byte, 16), nil)

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
	payload, _ := proto.Marshal(buildUpload(make([]byte, 16), nil, nil))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "server public key was not expected length", hook.LastEntry().Message)
	hook.Reset()

	// Public cert not found
	payload, _ = proto.Marshal(buildUpload(badPub[:], nil, nil))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 401, resp.Code, "401 response is expected")

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "failure to resolve client keypair", hook.LastEntry().Message)
	hook.Reset()

	// Nonce incorrect length
	payload, _ = proto.Marshal(buildUpload(goodPub[:], make([]byte, 16), nil))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "nonce was not expected length", hook.LastEntry().Message)
	hook.Reset()

	// App Public cert too short
	payload, _ = proto.Marshal(buildUpload(goodPub[:], make([]byte, 24), make([]byte, 16)))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code, "400 response is expected")

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Equal(t, "app public key key was not expected length", hook.LastEntry().Message)
	hook.Reset()

	// Server private cert too short
	payload, _ = proto.Marshal(buildUpload(goodPubBadPriv[:], make([]byte, 24), make([]byte, 32)))
	req, _ = http.NewRequest("POST", "/upload", bytes.NewReader(payload))
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 500, resp.Code, "500 response is expected")

	assert.Equal(t, 1, len(hook.Entries))
	assert.Equal(t, logrus.ErrorLevel, hook.LastEntry().Level)
	assert.Equal(t, "server private key was not expected length", hook.LastEntry().Message)
	hook.Reset()
}

func buildUpload(serverPubKey []byte, nonce []byte, appPublicKey []byte) *pb.EncryptedUploadRequest {
	upload := &pb.EncryptedUploadRequest{
		ServerPublicKey: serverPubKey,
		AppPublicKey:    appPublicKey,
		Nonce:           nonce,
		Payload:         make([]byte, 16),
	}
	return upload
}
