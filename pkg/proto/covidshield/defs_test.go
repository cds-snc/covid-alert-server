package covidshield

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIntoKey(t *testing.T) {

	expectedErr := errors.New("slice was not len=32")
	_, err := IntoKey([]byte{})
	assert.Equal(t, expectedErr, err, "slice should be 32 values long")

	expected := &[32]uint8{0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61}
	returned, _ := IntoKey([]byte(strings.Repeat("a", 32)))
	assert.Equal(t, expected, returned, "should return a 32 length slice")

}

func TestIntoNonce(t *testing.T) {

	expectedErr := errors.New("slice was not len=24")
	_, err := IntoNonce([]byte{})
	assert.Equal(t, expectedErr, err, "slice should be 24 values long")

	expected := &[24]uint8{0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61}
	returned, _ := IntoNonce([]byte(strings.Repeat("a", 24)))
	assert.Equal(t, expected, returned, "should return a 24 length slice")

}

func TestCurrentRollingStartIntervalNumber(t *testing.T) {

	epochTime := time.Now().Unix()
	intervalNumber := int32(epochTime / (60 * 10))
	expected := (intervalNumber / MaxTEKRollingPeriod) * MaxTEKRollingPeriod
	assert.Equal(t, expected, CurrentRollingStartIntervalNumber(), "should return current tolling interval number")

}
