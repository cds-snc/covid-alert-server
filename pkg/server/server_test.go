package server

import (
	"errors"
	"github.com/cds-snc/covid-alert-server/pkg/testhelpers"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/Shopify/goose/logger"
	"github.com/Shopify/goose/safely"
	"github.com/Shopify/goose/srvutil"
	pb "github.com/cds-snc/covid-alert-server/pkg/proto/covidshield"
	"github.com/cds-snc/covid-alert-server/pkg/telemetry"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"gopkg.in/tomb.v2"
)

func TestNew(t *testing.T) {

	// Returns a server struct
	bind := "0.0.0.0"

	servlets := make([]srvutil.Servlet, 0)
	sl := srvutil.CombineServlets(servlets...)

	sl = srvutil.UseServlet(sl,
		srvutil.RequestContextMiddleware,
		srvutil.RequestMetricsMiddleware,
		safely.Middleware,
		telemetry.OpenTelemetryMiddleware,
	)
	expectedResult := srvutil.NewServer(&tomb.Tomb{}, bind, sl)
	receivedResult := New(bind, servlets)
	assert.Equal(t, reflect.TypeOf(expectedResult), reflect.TypeOf(receivedResult), "Expected a new server struct")

}

func TestRequestError(t *testing.T) {
	// Capture logs
	oldLog := log
	defer func() { log = oldLog }()

	nullLog, hook := test.NewNullLogger()
	nullLog.ExitFunc = func(code int) {}

	log = func(ctx logger.Valuer, err ...error) *logrus.Entry {
		return logrus.NewEntry(nullLog)
	}

	req, _ := http.NewRequest("POST", "/", nil)
	ctx := req.Context()
	resp := httptest.NewRecorder()
	err := errors.New("Test")
	logMessage := "This is a message"

	pbError := pb.KeyClaimResponse_UNKNOWN
	msg := &pb.KeyClaimResponse{Error: &pbError}

	// Test StatusInternalServerError
	code := 500

	expectedResponse := result{}
	receivedResponse := requestError(ctx, resp, err, logMessage, code, msg)
	assert.Equal(t, expectedResponse, receivedResponse, "should return a result{} struct")

	testhelpers.AssertLog(t, hook, 1, logrus.ErrorLevel, "This is a message")

	// Test Warn
	code = 400

	expectedResponse = result{}
	receivedResponse = requestError(ctx, resp, err, logMessage, code, msg)
	assert.Equal(t, expectedResponse, receivedResponse, "should return a result{} struct")

	testhelpers.AssertLog(t, hook, 1, logrus.WarnLevel, "This is a message")

	// Test Headers
	code = 500

	expectedResponse = result{}
	receivedResponse = requestError(ctx, resp, err, logMessage, code, msg)
	assert.Equal(t, expectedResponse, receivedResponse, "should return a result{} struct")

	assert.Contains(t, resp.Header()["Content-Type"], "application/x-protobuf", "Content-Type should be set to application/x-protobuf")
	assert.Contains(t, resp.Header()["X-Content-Type-Options"], "nosniff", "Content-Type should be set to nosniff")
	assert.Equal(t, resp.Code, code, "status code should be set correctly")

	testhelpers.AssertLog(t, hook, 1, logrus.ErrorLevel, "This is a message")
}
