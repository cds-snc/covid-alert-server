package retrieval

import (
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockSigner "github.com/cds-snc/covid-alert-server/mocks/pkg/retrieval"
	pb "github.com/cds-snc/covid-alert-server/pkg/proto/covidshield"
	timestamp "github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSerializeOutbreakEventsTo(t *testing.T) {
	req, _ := http.NewRequest("POST", "/", nil)
	ctx := req.Context()
	resp := httptest.NewRecorder()
	locations := []*pb.OutbreakEvent{randomTestOutbreakEvent(), randomTestOutbreakEvent()}
	startTimestamp := time.Now()
	endTimestamp := time.Now().Add(1 * time.Hour)
	signer := &mockSigner.Signer{}

	data := make([]byte, 32)
	rand.Read(data)

	signer.On("Sign", mock.AnythingOfType("[]uint8")).Return(data, nil)

	expectedTotal := 164
	receivedTotal, receivedZip := SerializeOutbreakEventsTo(ctx, resp, locations, startTimestamp, endTimestamp, signer)

	assert.Equal(t, expectedTotal, receivedTotal)
	assert.Nil(t, receivedZip)
}

func randomTestOutbreakEvent() *pb.OutbreakEvent {
	uuid := "8a2c34b2-74a5-4b6a-8bed-79b7823b37c7"
	startTime, _ := timestamp.TimestampProto(time.Unix(1613238163, 0))
	endTime, _ := timestamp.TimestampProto(time.Unix(1613324563, 0))
	location := &pb.OutbreakEvent{LocationId: &uuid, StartTime: startTime, EndTime: endTime}
	return location
}
