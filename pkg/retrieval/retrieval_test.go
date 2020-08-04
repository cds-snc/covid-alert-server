package retrieval

import (
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockSigner "github.com/CovidShield/server/mocks/pkg/retrieval"
	pb "github.com/CovidShield/server/pkg/proto/covidshield"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMin(t *testing.T) {

	a := 5
	b := 6
	c := -1
	d := -6
	e := 0

	abExpected := a
	acExpected := c
	cdExpected := d
	deExpected := d
	aeExpected := e

	assert.Equal(t, abExpected, min(a, b))
	assert.Equal(t, acExpected, min(a, c))
	assert.Equal(t, cdExpected, min(c, d))
	assert.Equal(t, deExpected, min(d, e))
	assert.Equal(t, aeExpected, min(a, e))
}

func TestTransformRegion(t *testing.T) {

	reg := "302"
	regBadOtherInt := "1233"
	regBadString := "foo"

	ExpectedReg := "CA"
	ExpectedRegBadInt := regBadOtherInt
	ExpectedRegBadString := regBadString

	assert.Equal(t, ExpectedReg, transformRegion(reg))
	assert.Equal(t, ExpectedRegBadInt, transformRegion(regBadOtherInt))
	assert.Equal(t, ExpectedRegBadString, transformRegion(regBadString))
}

func TestSerializeTo(t *testing.T) {
	req, _ := http.NewRequest("POST", "/", nil)
	ctx := req.Context()
	resp := httptest.NewRecorder()
	region := "302"
	keys := []*pb.TemporaryExposureKey{randomTestKey(), randomTestKey()}
	startTimestamp := time.Now()
	endTimestamp := time.Now().Add(1 * time.Hour)
	signer := &mockSigner.Signer{}

	data := make([]byte, 32)
	rand.Read(data)

	signer.On("Sign", mock.AnythingOfType("[]uint8")).Return(data, nil)

	expectedTotal := 206
	receivedTotal, receivedZip := SerializeTo(ctx, resp, keys, region, startTimestamp, endTimestamp, signer)

	assert.Equal(t, expectedTotal, receivedTotal)
	assert.Nil(t, receivedZip)
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
