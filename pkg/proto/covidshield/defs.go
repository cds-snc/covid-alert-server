package covidshield

import (
	"errors"
	"github.com/CovidShield/server/pkg/config"
	"time"
)

const (
	// NonceLength is the length of a NaCl Box Nonce
	NonceLength = 24
	// KeyLength is the length of a NaCl Box Public or Private Key
	KeyLength = 32
	// KeyDataLength is the length of an Exposure Notification Temporary Exposure Key (Data)
	KeyDataLength = 16
)

func IntoKey(bytes []byte) (*[KeyLength]byte, error) {
	var arr [KeyLength]byte
	if len(bytes) != KeyLength {
		return nil, errors.New("slice was not len=32")
	}
	for i := 0; i < KeyLength; i++ {
		arr[i] = bytes[i]
	}
	return &arr, nil
}

func IntoNonce(bytes []byte) (*[NonceLength]byte, error) {
	var arr [NonceLength]byte
	if len(bytes) != NonceLength {
		return nil, errors.New("slice was not len=24")
	}
	for i := 0; i < NonceLength; i++ {
		arr[i] = bytes[i]
	}
	return &arr, nil
}

func CurrentRollingStartIntervalNumber() int32 {
	epochTime := time.Now().Unix()
	intervalNumber := int32(epochTime / (60 * 10))
	return (intervalNumber / int32(config.AppConstants.TEKRollingPeriod)) * int32(config.AppConstants.TEKRollingPeriod)
}
