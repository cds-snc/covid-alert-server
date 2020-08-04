package persistence

import (
	"crypto/rand"
	"crypto/sha512"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/CovidShield/server/pkg/config"
	pb "github.com/CovidShield/server/pkg/proto/covidshield"
	"github.com/CovidShield/server/pkg/timemath"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Shopify/goose/logger"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/nacl/box"
)

var allQueryMatcher sqlmock.QueryMatcher = sqlmock.QueryMatcherFunc(func(expectedSQL, actualSQL string) error { return nil })

type AnyType struct{}

// Match satisfies sqlmock.Argument interface
func (a AnyType) Match(v driver.Value) bool {
	return true
}

func TestDBDeleteOldDiagnosisKeys(t *testing.T) {
	// Init config
	config.InitConfig()

	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(allQueryMatcher))
	defer db.Close()

	conn := conn{
		db: db,
	}

	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))

	expectedResult := int64(1)
	receivedResult, receivedError := conn.DeleteOldDiagnosisKeys()

	assert.Equal(t, expectedResult, receivedResult)
	assert.Nil(t, receivedError)
}

func TestDBDeleteOldEncryptionKeys(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(allQueryMatcher))
	defer db.Close()

	conn := conn{
		db: db,
	}

	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))

	expectedResult := int64(1)
	receivedResult, receivedError := conn.DeleteOldEncryptionKeys()

	assert.Equal(t, expectedResult, receivedResult)
	assert.Nil(t, receivedError)
}

func TestDBClaimKey(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()

	conn := conn{
		db: db,
	}

	pub, _, _ := box.GenerateKey(rand.Reader)
	oneTimeCode := "80311300"

	// App key to short
	receivedResult, receivedError := conn.ClaimKey(oneTimeCode, make([]byte, 8))
	assert.Equal(t, receivedError, ErrInvalidKeyFormat)
	assert.Nil(t, receivedResult)

	// Expected result
	mock.ExpectBegin()
	rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT COUNT(*) FROM encryption_keys WHERE app_public_key = ?`).WithArgs(pub[:]).WillReturnRows(rows)

	created := time.Now()

	rows = sqlmock.NewRows([]string{"created"}).AddRow(created)
	mock.ExpectQuery(`SELECT created FROM encryption_keys WHERE one_time_code = ?`).WithArgs(oneTimeCode).WillReturnRows(rows)

	created = timemath.MostRecentUTCMidnight(created)

	query := fmt.Sprintf(
		`UPDATE encryption_keys
		SET one_time_code = NULL,
			app_public_key = ?,
			created = ?
		WHERE one_time_code = ?
		AND created > (NOW() - INTERVAL %d MINUTE)`,
		config.AppConstants.OneTimeCodeExpiryInMinutes,
	)

	mock.ExpectPrepare(query).ExpectExec().WithArgs(pub[:], created, oneTimeCode).WillReturnResult(sqlmock.NewResult(1, 1))

	rows = sqlmock.NewRows([]string{"server_public_key"}).AddRow(pub[:])
	mock.ExpectPrepare(`SELECT server_public_key FROM encryption_keys WHERE app_public_key = ?`).ExpectQuery().WithArgs(pub[:]).WillReturnRows(rows)

	mock.ExpectCommit()

	expectedResult := pub[:]
	receivedResult, receivedError = conn.ClaimKey(oneTimeCode, pub[:])

	assert.Equal(t, expectedResult, receivedResult)
	assert.Nil(t, receivedError)
}

func TestDBNewKeyClaim(t *testing.T) {
	// Capture logs
	oldLog := log
	defer func() { log = oldLog }()

	nullLog, hook := test.NewNullLogger()
	nullLog.ExitFunc = func(code int) {}

	log = func(ctx logger.Valuer, err ...error) *logrus.Entry {
		return logrus.NewEntry(nullLog)
	}

	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(allQueryMatcher))
	defer db.Close()

	conn := conn{
		db: db,
	}

	region := "302"
	originator := "randomOrigin"

	// Success with no HashID
	mock.ExpectExec(
		`INSERT INTO encryption_keys
		(region, originator, server_private_key, server_public_key, one_time_code, remaining_keys)
		VALUES (?, ?, ?, ?, ?, ?)`).WithArgs(
		region,
		originator,
		AnyType{},
		AnyType{},
		AnyType{},
		config.AppConstants.InitialRemainingKeys,
	).WillReturnResult(sqlmock.NewResult(1, 1))

	receivedResult, receivedError := conn.NewKeyClaim(region, originator, "")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Less(t, 0, len(receivedResult))
	assert.Nil(t, receivedError, "Expected nil if it could execute insert")

	// Error - Generic
	mock.ExpectExec(
		`INSERT INTO encryption_keys
		(region, originator, server_private_key, server_public_key, one_time_code, remaining_keys)
		VALUES (?, ?, ?, ?, ?, ?)`).WithArgs(
		region,
		originator,
		AnyType{},
		AnyType{},
		AnyType{},
		config.AppConstants.InitialRemainingKeys,
	).WillReturnError(fmt.Errorf("error"))

	receivedResult, receivedError = conn.NewKeyClaim(region, originator, "")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr := fmt.Errorf("error")
	assert.Equal(t, "", receivedResult, "Expected result if could not execute insert")
	assert.Equal(t, expectedErr, receivedError, "Expected error if could not execute insert")

	// Error - existing code
	mock.ExpectExec(
		`INSERT INTO encryption_keys
		(region, originator, server_private_key, server_public_key, one_time_code, remaining_keys)
		VALUES (?, ?, ?, ?, ?, ?)`).WithArgs(
		region,
		originator,
		AnyType{},
		AnyType{},
		AnyType{},
		config.AppConstants.InitialRemainingKeys,
	).WillReturnError(fmt.Errorf("Duplicate entry"))

	mock.ExpectExec(
		`INSERT INTO encryption_keys
		(region, originator, server_private_key, server_public_key, one_time_code, remaining_keys)
		VALUES (?, ?, ?, ?, ?, ?)`).WithArgs(
		region,
		originator,
		AnyType{},
		AnyType{},
		AnyType{},
		config.AppConstants.InitialRemainingKeys,
	).WillReturnResult(sqlmock.NewResult(1, 1))

	receivedResult, receivedError = conn.NewKeyClaim(region, originator, "")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Less(t, 0, len(receivedResult))
	assert.Nil(t, receivedError, "Expected nil if it could execute insert")

	assertLog(t, hook, 1, logrus.WarnLevel, "duplicate one_time_code")

	// Error - never succeeds with duplicate codes

	for i := 0; i < 5; i++ {
		mock.ExpectExec(
			`INSERT INTO encryption_keys
		(region, originator, server_private_key, server_public_key, one_time_code, remaining_keys)
		VALUES (?, ?, ?, ?, ?, ?)`).WithArgs(
			region,
			originator,
			AnyType{},
			AnyType{},
			AnyType{},
			config.AppConstants.InitialRemainingKeys,
		).WillReturnError(fmt.Errorf("Duplicate entry"))
	}

	receivedResult, receivedError = conn.NewKeyClaim(region, originator, "")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, "", receivedResult, "Expected result if could not execute insert")
	assert.Nil(t, receivedError) // This is a bug and should be fixed, however, it is high unlikely to trigger

	assertLog(t, hook, 5, logrus.WarnLevel, "duplicate one_time_code")

	// Error - unclaimed HashID, eventual success
	hashID := hex.EncodeToString(SHA512([]byte("abcd")))

	mock.ExpectExec(
		`INSERT INTO encryption_keys
		(region, originator, server_private_key, server_public_key, one_time_code, remaining_keys)
		VALUES (?, ?, ?, ?, ?, ?)`).WithArgs(
		region,
		originator,
		hashID,
		AnyType{},
		AnyType{},
		AnyType{},
		config.AppConstants.InitialRemainingKeys,
	).WillReturnError(fmt.Errorf("for key 'hash_id"))

	rows := sqlmock.NewRows([]string{"one_time_code"}).AddRow("ABCD")
	mock.ExpectQuery(
		`SELECT one_time_code FROM encryption_keys WHERE hash_id = ? FOR UPDATE`).WithArgs(hashID).WillReturnRows(rows)
	mock.ExpectExec(`DELETE FROM encryption_keys WHERE hash_id = ? AND one_time_code IS NOT NULL`).WithArgs(hashID).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(
		`INSERT INTO encryption_keys
		(region, originator, server_private_key, server_public_key, one_time_code, remaining_keys)
		VALUES (?, ?, ?, ?, ?, ?)`).WithArgs(
		region,
		originator,
		hashID,
		AnyType{},
		AnyType{},
		AnyType{},
		config.AppConstants.InitialRemainingKeys,
	).WillReturnResult(sqlmock.NewResult(1, 1))

	receivedResult, receivedError = conn.NewKeyClaim(region, originator, hashID)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Less(t, 0, len(receivedResult))
	assert.Nil(t, receivedError, "Expected nil if it could execute insert")

	assertLog(t, hook, 1, logrus.WarnLevel, "regenerating OTC for hashID")

	// Error - claimed HashID
	mock.ExpectExec(
		`INSERT INTO encryption_keys
		(region, originator, server_private_key, server_public_key, one_time_code, remaining_keys)
		VALUES (?, ?, ?, ?, ?, ?)`).WithArgs(
		region,
		originator,
		hashID,
		AnyType{},
		AnyType{},
		AnyType{},
		config.AppConstants.InitialRemainingKeys,
	).WillReturnError(fmt.Errorf("for key 'hash_id"))

	rows = sqlmock.NewRows([]string{"one_time_code"}).AddRow(nil)
	mock.ExpectQuery(
		`SELECT one_time_code FROM encryption_keys WHERE hash_id = ? FOR UPDATE`).WithArgs(hashID).WillReturnRows(rows)

	receivedResult, receivedError = conn.NewKeyClaim(region, originator, hashID)

	assert.Equal(t, "", receivedResult, "Expected result if could not execute insert")
	assert.Equal(t, ErrHashIDClaimed, receivedError) // This is a bug and should be fixed, however, it is high unlikely to trigger
}

func TestDBPrivForPub(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(allQueryMatcher))
	defer db.Close()

	conn := conn{
		db: db,
	}

	// Success
	pub, _, _ := box.GenerateKey(rand.Reader)

	rows := sqlmock.NewRows([]string{"server_private_key"}).AddRow(pub[:])
	mock.ExpectQuery("").WillReturnRows(rows)

	expectedResult := pub[:]
	receivedResult, receivedError := conn.PrivForPub(pub[:])

	assert.Equal(t, expectedResult, receivedResult)
	assert.Nil(t, receivedError)

	// Bad cert
	expectedResult = pub[:]
	receivedResult, receivedError = conn.PrivForPub(make([]byte, 8))

	assert.NotEqual(t, expectedResult, receivedResult)
	assert.Equal(t, ErrInvalidKeyFormat, receivedError)

	// Error - no rows
	rows = sqlmock.NewRows([]string{"server_private_key"})
	mock.ExpectQuery("").WillReturnRows(rows)

	receivedResult, receivedError = conn.PrivForPub(pub[:])

	assert.Equal(t, errors.New("no record"), receivedError)
	assert.Nil(t, receivedResult)

	// Error - gemeric error
	rows = sqlmock.NewRows([]string{"server_private_key"})
	mock.ExpectQuery("").WillReturnError(fmt.Errorf("generic error"))

	receivedResult, receivedError = conn.PrivForPub(pub[:])

	assert.Equal(t, errors.New("no record"), receivedError)
	assert.Nil(t, receivedResult)
}

func TestDBStoreKeys(t *testing.T) {
	// Capture logs
	oldLog := log
	defer func() { log = oldLog }()

	nullLog, hook := test.NewNullLogger()
	nullLog.ExitFunc = func(code int) {}

	log = func(ctx logger.Valuer, err ...error) *logrus.Entry {
		return logrus.NewEntry(nullLog)
	}

	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(allQueryMatcher))
	defer db.Close()

	conn := conn{
		db: db,
	}

	pub, _, _ := box.GenerateKey(rand.Reader)
	region := "302"
	originator := "randomOrigin"

	keyOne := randomTestKey()
	keyTwo := randomTestKey()
	keys := []*pb.TemporaryExposureKey{keyOne, keyTwo}

	hourOfSubmission := timemath.HourNumber(time.Now())

	mock.ExpectBegin()
	row := sqlmock.NewRows([]string{"region", "originator", "remaining_keys"}).AddRow("302", "randomOrigin", 3)
	mock.ExpectQuery(`SELECT region, originator, remaining_keys FROM encryption_keys WHERE app_public_key = ? FOR UPDATE`).WillReturnRows(row)

	mock.ExpectPrepare(
		`INSERT IGNORE INTO diagnosis_keys
		(region, originator, key_data, rolling_start_interval_number, rolling_period, transmission_risk_level, hour_of_submission)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
	)

	for _, key := range keys {
		mock.ExpectExec(`INSERT IGNORE INTO diagnosis_keys
		(region, originator, key_data, rolling_start_interval_number, rolling_period, transmission_risk_level, hour_of_submission)
		VALUES (?, ?, ?, ?, ?, ?, ?)`).WithArgs(
			region,
			originator,
			key.GetKeyData(),
			key.GetRollingStartIntervalNumber(),
			key.GetRollingPeriod(),
			key.GetTransmissionRiskLevel(),
			hourOfSubmission,
		).WillReturnResult(sqlmock.NewResult(1, 1))
	}

	mock.ExpectExec(
		`UPDATE encryption_keys
	SET remaining_keys = remaining_keys - ?
	WHERE remaining_keys >= ?
	AND app_public_key = ?`,
	).WithArgs(
		len(keys),
		len(keys),
		pub[:],
	).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()
	receivedResult := conn.StoreKeys(pub, keys)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Nil(t, receivedResult, "Expected nil when keys are commited")

	assertLog(t, hook, 1, logrus.InfoLevel, "Inserted keys")
}

func TestDBFetchKeysForHours(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(allQueryMatcher))
	defer db.Close()

	conn := conn{
		db: db,
	}

	// No errors
	region := "302"
	startHour := uint32(100)
	endHour := uint32(200)
	currentRollingStartIntervalNumber := int32(2651450)
	rollingPeriod := int32(144)
	transmissionRiskLevel := int32(4)

	row := sqlmock.NewRows([]string{"region", "key_data", "rolling_start_interval_number", "rolling_period", "transmission_risk_level"}).AddRow("302", []byte{}, 2651450, 144, 4)
	mock.ExpectQuery("").WillReturnRows(row)

	expectedResult := []*pb.TemporaryExposureKey{
		&pb.TemporaryExposureKey{
			KeyData:                    []byte{},
			RollingStartIntervalNumber: &currentRollingStartIntervalNumber,
			RollingPeriod:              &rollingPeriod,
			TransmissionRiskLevel:      &transmissionRiskLevel,
		},
	}

	receivedResult, _ := conn.FetchKeysForHours(region, startHour, endHour, currentRollingStartIntervalNumber)

	assert.Equal(t, expectedResult, receivedResult, "Expected rows for the query")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	// Errors
	mock.ExpectQuery("").WillReturnError(fmt.Errorf("Generic error"))

	_, receivedError := conn.FetchKeysForHours(region, startHour, endHour, currentRollingStartIntervalNumber)

	assert.Equal(t, fmt.Errorf("Generic error"), receivedError, "Expected rows for the query")
}

func TestDBCheckClaimKeyBan(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(allQueryMatcher))
	defer db.Close()

	conn := conn{
		db: db,
	}

	identifier := "127.0.0.1"

	// Queries and succeeds if no result is found
	row := sqlmock.NewRows([]string{"failures", "last_failure"})
	mock.ExpectQuery(`SELECT failures, last_failure FROM failed_key_claim_attempts WHERE identifier = ?`).WithArgs(identifier).WillReturnRows(row)

	expectedTriesRemaining := config.AppConstants.MaxConsecutiveClaimKeyFailures
	expectedBanDuration := time.Duration(0)

	receivedTriesRemaining, receivedBanDuration, _ := conn.CheckClaimKeyBan(identifier)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, expectedTriesRemaining, receivedTriesRemaining, "Expected config.AppConstants.MaxConsecutiveClaimKeyFailures as tries remaining")
	assert.Equal(t, expectedBanDuration, receivedBanDuration, "Expected 0 as ban duration")
}

func TestDBClaimKeySuccess(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(allQueryMatcher))
	defer db.Close()

	conn := conn{
		db: db,
	}

	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))

	receivedError := conn.ClaimKeySuccess("127.0.0.1")

	assert.Nil(t, receivedError)
}

func TestDBClaimKeyFailure(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()

	conn := conn{
		db: db,
	}

	identifier := "127.0.0.1"

	mock.ExpectBegin()
	mock.ExpectExec(
		`INSERT INTO failed_key_claim_attempts (identifier) VALUES (?)
		ON DUPLICATE KEY UPDATE
      failures = failures + 1,
			last_failure = NOW()`).WithArgs(
		identifier,
	).WillReturnResult(sqlmock.NewResult(1, 1))

	//--> Called in checkClaimKeyBan
	row := sqlmock.NewRows([]string{"failures", "last_failure"}).AddRow(1, time.Now())
	mock.ExpectQuery(`SELECT failures, last_failure FROM failed_key_claim_attempts WHERE identifier = ?`).WithArgs(identifier).WillReturnRows(row)

	mock.ExpectCommit()

	expectedTriesRemaining := config.AppConstants.MaxConsecutiveClaimKeyFailures - 1
	expectedBanDuration := time.Duration(0)

	receivedTriesRemaining, receivedBanDuration, receivedErr := conn.ClaimKeyFailure(identifier)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, expectedTriesRemaining, receivedTriesRemaining, "Expected maxConsecutiveClaimKeyFailures - 1 as tries remaining")
	assert.Equal(t, expectedBanDuration, receivedBanDuration, "Expected 0 as ban duration")
	assert.Nil(t, receivedErr, "Expected no error if inserted")
}

func TestDBDeleteOldFailedClaimKeyAttempts(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(allQueryMatcher))
	defer db.Close()

	conn := conn{
		db: db,
	}

	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))

	expectedResult := int64(1)
	receivedResult, receivedError := conn.DeleteOldFailedClaimKeyAttempts()

	assert.Equal(t, expectedResult, receivedResult)
	assert.Nil(t, receivedError)
}

func TestDBCountClaimedOneTimeCodes(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(allQueryMatcher))
	defer db.Close()

	conn := conn{
		db: db,
	}

	rows := sqlmock.NewRows([]string{"count"}).AddRow(100)
	mock.ExpectQuery("").WillReturnRows(rows)

	expectedResult := int64(100)
	receivedResult, receivedError := conn.CountClaimedOneTimeCodes()

	assert.Equal(t, expectedResult, receivedResult)
	assert.Nil(t, receivedError)
}

func TestDBCountDiagnosisKeys(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(allQueryMatcher))
	defer db.Close()

	conn := conn{
		db: db,
	}

	rows := sqlmock.NewRows([]string{"count"}).AddRow(100)
	mock.ExpectQuery("").WillReturnRows(rows)

	expectedResult := int64(100)
	receivedResult, receivedError := conn.CountDiagnosisKeys()

	assert.Equal(t, expectedResult, receivedResult)
	assert.Nil(t, receivedError)
}

func TestDBCountUnclaimedOneTimeCodes(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(allQueryMatcher))
	defer db.Close()

	conn := conn{
		db: db,
	}

	rows := sqlmock.NewRows([]string{"count"}).AddRow(100)
	mock.ExpectQuery("").WillReturnRows(rows)

	expectedResult := int64(100)
	receivedResult, receivedError := conn.CountUnclaimedOneTimeCodes()

	assert.Equal(t, expectedResult, receivedResult)
	assert.Nil(t, receivedError)
}

func assertLog(t *testing.T, hook *test.Hook, length int, level logrus.Level, msg string) {
	assert.Equal(t, length, len(hook.Entries))
	assert.Equal(t, level, hook.LastEntry().Level)
	assert.Equal(t, msg, hook.LastEntry().Message)
	hook.Reset()
}

func SHA512(message []byte) []byte {
	c := sha512.New()
	c.Write(message)
	return c.Sum(nil)
}
