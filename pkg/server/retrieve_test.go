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

	"github.com/stretchr/testify/assert"
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

func TestFail(t *testing.T) {
}

func TestRetrieveWrapper(t *testing.T) {
}

func TestRetrieve(t *testing.T) {

	// Init config
	config.InitConfig()

	db := &persistence.Conn{}
	auth := &retrieval.Authenticator{}
	signer := &retrieval.Signer{}

	region := "302"
	goodAuth := "abcd"
	badAuth := "dcba"
	currentRSIN := pb.CurrentRollingStartIntervalNumber()
	currentDateNumber := fmt.Sprint(timemath.CurrentDateNumber())

	// Bad Auth Mock
	auth.On("Authenticate", region, currentDateNumber, badAuth).Return(false)

	// Good auth Mock
	auth.On("Authenticate", region, currentDateNumber, goodAuth).Return(true)

	// All keys download Mock
	auth.On("Authenticate", region, "00000", goodAuth).Return(true)
	startHour := (timemath.CurrentDateNumber() - 15) * 24
	endHour := timemath.CurrentDateNumber() * 24

	db.On("FetchKeysForHours", region, startHour, endHour, currentRSIN).Return([]*pb.TemporaryExposureKey{randomTestKey(), randomTestKey()}, nil)

	servlet := NewRetrieveServlet(db, auth, signer)
	router := Router()
	servlet.RegisterRouting(router)

	// Bad authentication
	req, _ := http.NewRequest("GET", fmt.Sprintf("/retrieve/%s/%s/%s", region, currentDateNumber, badAuth), nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 401, resp.Code, "Unauthorized response is expected")
	assert.Equal(t, "unauthorized\n", string(resp.Body.Bytes()), "Correct response is expected")

	// Bad Method
	req, _ = http.NewRequest("POST", fmt.Sprintf("/retrieve/%s/%s/%s", region, currentDateNumber, goodAuth), nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 405, resp.Code, "Method not allowed response is expected")
	assert.Equal(t, "method not allowed\n", string(resp.Body.Bytes()), "Correct response is expected")

	// Get all keys
	req, _ = http.NewRequest("GET", fmt.Sprintf("/retrieve/%s/%s/%s", region, "00000", goodAuth), nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, 405, resp.Code, "Method not allowed response is expected")
	assert.Equal(t, "method not allowed\n", string(resp.Body.Bytes()), "Correct response is expected")

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
