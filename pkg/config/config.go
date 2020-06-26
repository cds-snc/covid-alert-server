package config

import (
	"fmt"
	"github.com/spf13/viper"
)

type Constants struct {
	DefaultSubmissionServerPort    uint32
	DefaultRetrievalServerPort     uint32
	DefaultServerPort              uint32
	WorkerExpirationInterval       uint32
	MaxOneTimeCode                 int64
	MaxConsecutiveClaimKeyFailures int
	ClaimKeyBanDuration            uint32
	MaxDiagnosisKeyRetentionDays   uint32
	InitialRemainingKeys           uint32
	EncryptionKeyValidityDays      uint32
	OneTimeCodeExpiryInMinutes     uint32
	AssignmentParts                int
	HmacKeyLength                  int
	TEKRollingPeriod               int
	MaxKeysInUpload                int
}

var AppConstants Constants

func InitConfig() {
	viper.SetConfigName("config")
	viper.AddConfigPath("../../")
	viper.SetConfigType("yaml")
	setDefaults()
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file, %s", err)
	}
	err := viper.Unmarshal(&AppConstants)
	if err != nil {
		fmt.Printf("Unable to unmarshal the config file, %v", err)
	}
}

func setDefaults() {
	viper.SetDefault("defaultSubmissionServerPort", 8000)
	viper.SetDefault("defaultRetrievalServerPort", 8001)
	viper.SetDefault("defaultServerPort", 8010)
	viper.SetDefault("expirationInterval", 30)
	viper.SetDefault("maxOneTimeCode", 1e8)
	viper.SetDefault("maxConsecutiveClaimKeyFailures", 8)
	viper.SetDefault("claimKeyBanDuration", 1)
	viper.SetDefault("maxDiagnosisKeyRetentionDays", 15)
	viper.SetDefault("initialRemainingKeys", 28)
	viper.SetDefault("encryptionKeyValidityDays", 15)
	viper.SetDefault("oneTimeCodeExpiryInMinutes", 1440)
	viper.SetDefault("assignmentParts", 2)
	viper.SetDefault("hmacKeyLength", 32)
	viper.SetDefault("tekRollingPeriod", 144)
	viper.SetDefault("maxKeysInUpload", 14)
}
