package timemath

import (
	"testing"
	"time"

	pb "github.com/CovidShield/server/pkg/proto/covidshield"
	"github.com/stretchr/testify/assert"
)

func TestHourNumber(t *testing.T) {

	expected := uint32(time.Now().Unix() / 3600)
	assert.Equal(t, expected, HourNumber(time.Now()))

}

func TestDateNumber(t *testing.T) {

	expected := uint32(time.Now().Unix() / 86400)
	assert.Equal(t, expected, DateNumber(time.Now()))

}

func TestMostRecentUTCMidnight(t *testing.T) {

	expected := time.Unix((time.Now().UTC().Unix()/SecondsInDay)*SecondsInDay, 0).UTC()
	assert.Equal(t, expected, MostRecentUTCMidnight(time.Now()))

}

func TestHourNumberAtStartOfDate(t *testing.T) {

	expected := uint32(1000 * 24)
	assert.Equal(t, expected, HourNumberAtStartOfDate(1000))

}

func TestHourNumberPlusDays(t *testing.T) {

	expected := uint32(int(20000) + 24*10)
	assert.Equal(t, expected, HourNumberPlusDays(20000, 10))

}

func TestRollingStartIntervalNumberPlusDays(t *testing.T) {

	expected := int32(int(20000) + 10*pb.MaxTEKRollingPeriod)
	assert.Equal(t, expected, RollingStartIntervalNumberPlusDays(20000, 10))

}

func TestCurrentDateNumber(t *testing.T) {

	expected := DateNumber(time.Now())
	assert.Equal(t, expected, CurrentDateNumber())

}
