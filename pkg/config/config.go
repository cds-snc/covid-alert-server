package config

import (
	"flag"

	"github.com/Shopify/goose/logger"
	"github.com/spf13/viper"
)

var log = logger.New("config")

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
}

var AppConstants Constants

func InitConfig() {
	viper.SetConfigName("config")
	// Reading config file path from command line flag
	configFilePath := flag.String("config_file_path", "../", "Path for Viper config.yaml")
	flag.Parse()
	viper.AddConfigPath(*configFilePath)
	viper.SetConfigType("yaml")
	setDefaults()
	if err := viper.ReadInConfig(); err != nil {
		log(nil, err).Fatal("Error reading application configuration file")
	}
	err := viper.Unmarshal(&AppConstants)
	if err != nil {
		log(nil, err).Fatal("Unable to unmarshal the application configuration file")
	}
}

func setDefaults() {
	viper.SetDefault("defaultSubmissionServerPort", 8000)
	viper.SetDefault("defaultRetrievalServerPort", 8001)
	viper.SetDefault("defaultServerPort", 8010)
	viper.SetDefault("workerExpirationInterval", 30)
	viper.SetDefault("maxOneTimeCode", 1e8)
	viper.SetDefault("maxConsecutiveClaimKeyFailures", 8)
	viper.SetDefault("claimKeyBanDuration", 1)
	viper.SetDefault("maxDiagnosisKeyRetentionDays", 15)
	viper.SetDefault("initialRemainingKeys", 28)
	viper.SetDefault("encryptionKeyValidityDays", 15)
	viper.SetDefault("oneTimeCodeExpiryInMinutes", 1440)
	viper.SetDefault("assignmentParts", 2)
	viper.SetDefault("hmacKeyLength", 32)
}
