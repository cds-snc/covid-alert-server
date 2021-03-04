package persistence

import (
	"context"
	"crypto/rand"
	"crypto/sha512"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Shopify/goose/logger"
	"github.com/cds-snc/covid-alert-server/pkg/config"
	pb "github.com/cds-snc/covid-alert-server/pkg/proto/covidshield"
	"github.com/cds-snc/covid-alert-server/pkg/timemath"
	timestamp "github.com/golang/protobuf/ptypes"
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

func TestDBDeleteOldExpiredKeys(t *testing.T) {

	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(allQueryMatcher))
	defer db.Close()

	conn := conn{
		db: db,
	}

	mock.ExpectBegin()
	rows := sqlmock.NewRows([]string{"originator", "count"}).AddRow("foo", 1).AddRow("bar", 2)
	mock.ExpectQuery(fmt.Sprintf(`SELECT originator, COUNT(*) FROM encryption_keys
		WHERE  (created < (NOW() - INTERVAL %d DAY))
		GROUP BY encryption_keys.originator`,
		config.AppConstants.EncryptionKeyValidityDays),
	).WillReturnRows(rows)
	mock.ExpectQuery(fmt.Sprintf(`SELECT originator, COUNT(*) FROM encryption_keys
		WHERE  (created < (NOW() - INTERVAL %d DAY)) AND remaining_keys = %d
		GROUP BY encryption_keys.originator`,
		config.AppConstants.OneTimeCodeExpiryInMinutes,
		config.AppConstants.InitialRemainingKeys),
	).WillReturnRows(rows)
	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	expectedResult := int64(1)
	receivedResult, receivedError := conn.DeleteExpiredKeys(context.Background())

	assert.Equal(t, expectedResult, receivedResult)
	assert.Nil(t, receivedError)

}

func TestDBDeleteOldUnclaimedKeys(t *testing.T) {

	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(allQueryMatcher))
	defer db.Close()

	conn := conn{
		db: db,
	}

	mock.ExpectBegin()
	rows := sqlmock.NewRows([]string{"originator", "count"}).AddRow("foo", 1).AddRow("bar", 2)

	mock.ExpectQuery(fmt.Sprintf(`
		SELECT originator, count(*) FROM encryption_keys
		WHERE  ((created < (NOW() - INTERVAL %d MINUTE)) AND app_public_key IS NULL)
		GROUP BY encryption_keys.originator `,
		config.AppConstants.OneTimeCodeExpiryInMinutes),
	).WillReturnRows(rows)
	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	expectedResult := int64(1)
	receivedResult, receivedError := conn.DeleteUnclaimedKeys(context.Background())

	assert.Equal(t, expectedResult, receivedResult)
	assert.Nil(t, receivedError)

}

func TestDBDeleteOldExhaustedKeys(t *testing.T) {

	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(allQueryMatcher))
	defer db.Close()

	conn := conn{
		db: db,
	}

	mock.ExpectBegin()
	rows := sqlmock.NewRows([]string{"originator", "count"}).AddRow("foo", 1).AddRow("bar", 2)
	mock.ExpectQuery(`
		SELECT originator, COUNT(*) FROM encryption_keys
		WHERE  remaining_keys = 0
		GROUP BY encryption_keys.originator`,
	).WillReturnRows(rows)
	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	expectedResult := int64(1)
	receivedResult, receivedError := conn.DeleteExhaustedKeys(context.Background())

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
	receivedResult, receivedError := conn.ClaimKey(oneTimeCode, make([]byte, 8), nil)
	assert.Equal(t, receivedError, ErrInvalidKeyFormat)
	assert.Nil(t, receivedResult)

	// Expected result
	mock.ExpectBegin()
	rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT COUNT(*) FROM encryption_keys WHERE app_public_key = ?`).WithArgs(pub[:]).WillReturnRows(rows)

	created := time.Now()
	originator := "onAPI"
	rows = sqlmock.NewRows([]string{"created", "originator"}).AddRow(created, originator)
	mock.ExpectQuery(`SELECT created, originator FROM encryption_keys WHERE one_time_code = ?`).WithArgs(oneTimeCode).WillReturnRows(rows)

	created = timemath.MostRecentUTCMidnight(created)

	query := `UPDATE encryption_keys
		SET one_time_code = NULL,
			app_public_key = ?,
			created = ?
		WHERE one_time_code = ?
		AND created > (NOW() - INTERVAL ? MINUTE)`

	mock.ExpectPrepare(query).
		ExpectExec().
		WithArgs(
			pub[:],
			created,
			oneTimeCode,
			config.AppConstants.OneTimeCodeExpiryInMinutes,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	rows = sqlmock.NewRows([]string{"server_public_key"}).AddRow(pub[:])
	mock.ExpectPrepare(`SELECT server_public_key FROM encryption_keys WHERE app_public_key = ?`).ExpectQuery().WithArgs(pub[:]).WillReturnRows(rows)

	mock.ExpectCommit()

	expectedResult := pub[:]
	receivedResult, receivedError = conn.ClaimKey(oneTimeCode, pub[:], nil)

	assert.Equal(t, expectedResult, receivedResult)
	assert.Nil(t, receivedError)
}

const (
	region     string = "302"
	originator string = "randomOrigin"
)

func TestSuccessWithNoHashID(t *testing.T) {
	// Capture logs
	oldLog := log
	defer func() { log = oldLog }()

	nullLog, _ := test.NewNullLogger()
	nullLog.ExitFunc = func(code int) {}

	log = func(ctx logger.Valuer, err ...error) *logrus.Entry {
		return logrus.NewEntry(nullLog)
	}

	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(allQueryMatcher))
	defer db.Close()

	conn := conn{
		db: db,
	}

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

	mock.ExpectBegin()
	setupSaveEventMock(mock, Event{
		Identifier: OTKGenerated,
		DeviceType: Server,
		Date:       time.Now(),
		Count:      1,
		Originator: originator,
	})
	mock.ExpectCommit()

	receivedResult, receivedError := conn.NewKeyClaim(context.TODO(), region, originator, "")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Less(t, 0, len(receivedResult))
	assert.Nil(t, receivedError, "Expected nil if it could execute insert")
}

func TestErrorGeneric(t *testing.T) {

	// Capture logs
	oldLog := log
	defer func() { log = oldLog }()

	nullLog, _ := test.NewNullLogger()
	nullLog.ExitFunc = func(code int) {}

	log = func(ctx logger.Valuer, err ...error) *logrus.Entry {
		return logrus.NewEntry(nullLog)
	}

	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(allQueryMatcher))
	defer db.Close()

	conn := conn{
		db: db,
	}
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

	receivedResult, receivedError := conn.NewKeyClaim(context.TODO(), region, originator, "")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr := fmt.Errorf("error")
	assert.Equal(t, "", receivedResult, "Expected result if could not execute insert")
	assert.Equal(t, expectedErr, receivedError, "Expected error if could not execute insert")
}

func TestErrorExistingCode(t *testing.T) {

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

	mock.ExpectBegin()
	setupSaveEventMock(mock,
		Event{
			Identifier: OTKGenerated,
			DeviceType: Server,
			Date:       time.Time{},
			Count:      1,
			Originator: "randomOrigin",
		})
	mock.ExpectCommit()

	receivedResult, receivedError := conn.NewKeyClaim(context.TODO(), region, originator, "")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Less(t, 0, len(receivedResult))
	assert.Nil(t, receivedError, "Expected nil if it could execute insert")

	assertLog(t, hook, 1, logrus.WarnLevel, "duplicate one_time_code")
}

func TestNeverSucceedsWithDuplicateCodes(t *testing.T) {

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

	for i := 0; i < 5; i++ {
		mock.ExpectExec(
			`INSERT INTO encryption_keys
		(region, originator, server_private_key, server_public_key, one_time_code, remaining_keys)
		VALUES (?, ?, ?, ?, ?, ?)`).WithArgs(
			"302",
			originator,
			AnyType{},
			AnyType{},
			AnyType{},
			config.AppConstants.InitialRemainingKeys,
		).WillReturnError(fmt.Errorf("Duplicate entry"))
	}

	receivedResult, receivedError := conn.NewKeyClaim(context.TODO(), region, originator, "")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, "", receivedResult, "Expected result if could not execute insert")
	assert.Nil(t, receivedError) // This is a bug and should be fixed, however, it is high unlikely to trigger

	assertLog(t, hook, 5, logrus.WarnLevel, "duplicate one_time_code")
}

func TestUnclaimedHashIDEventualSuccess(t *testing.T) {

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

	mock.ExpectBegin()
	setupSaveEventMock(mock, Event{
		Identifier: OTKRegenerated,
		DeviceType: Server,
		Date:       time.Now(),
		Count:      1,
		Originator: "randomOrigin",
	})
	mock.ExpectCommit()

	receivedResult, receivedError := conn.NewKeyClaim(context.TODO(), region, originator, hashID)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Less(t, 0, len(receivedResult))
	assert.Nil(t, receivedError, "Expected nil if it could execute insert")

	assertLog(t, hook, 1, logrus.WarnLevel, "regenerating OTC for hashID")
}

func TestClaimedHashID(t *testing.T) {
	// Capture logs
	oldLog := log
	defer func() { log = oldLog }()

	nullLog, _ := test.NewNullLogger()
	nullLog.ExitFunc = func(code int) {}

	log = func(ctx logger.Valuer, err ...error) *logrus.Entry {
		return logrus.NewEntry(nullLog)
	}

	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(allQueryMatcher))
	defer db.Close()

	conn := conn{
		db: db,
	}

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

	rows := sqlmock.NewRows([]string{"one_time_code"}).AddRow(nil)
	mock.ExpectQuery(
		`SELECT one_time_code FROM encryption_keys WHERE hash_id = ? FOR UPDATE`).WithArgs(hashID).WillReturnRows(rows)

	receivedResult, receivedError := conn.NewKeyClaim(context.TODO(), region, originator, hashID)

	assert.Equal(t, "", receivedResult, "Expected result if could not execute insert")
	assert.Equal(t, ErrHashIDClaimed, receivedError) // This is a bug and should be fixed, however, it is high unlikely to trigger
}

func TestNewOutbreakEventError(t *testing.T) {
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

	locationID := "ABCDEFGH"
	startTime, _ := timestamp.TimestampProto(time.Now())
	endTime, _ := timestamp.TimestampProto(time.Now())
	submission := pb.OutbreakEvent{LocationId: &locationID, StartTime: startTime, EndTime: endTime}

	mock.ExpectExec(
		`INSERT INTO qr_outbreak_events
		(location_id, originator, start_time, end_time, severity)
		VALUES (?, ?, ?, ?, ?)`).WithArgs(
		AnyType{},
		originator,
		AnyType{},
		AnyType{},
		AnyType{},
	).WillReturnError(fmt.Errorf("error"))

	receivedError := conn.NewOutbreakEvent(context.TODO(), originator, &submission)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr := fmt.Errorf("error")
	assert.Equal(t, expectedErr, receivedError, "Expected error if could not execute insert")
	assertLog(t, hook, 1, logrus.ErrorLevel, "saving new QR submission")
}

func TestNewOutbreakEventSuccess(t *testing.T) {
	// Capture logs
	oldLog := log
	defer func() { log = oldLog }()

	nullLog, _ := test.NewNullLogger()
	nullLog.ExitFunc = func(code int) {}

	log = func(ctx logger.Valuer, err ...error) *logrus.Entry {
		return logrus.NewEntry(nullLog)
	}

	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(allQueryMatcher))
	defer db.Close()

	conn := conn{
		db: db,
	}

	locationID := "ABCDEFGH"
	startTime, _ := timestamp.TimestampProto(time.Now())
	endTime, _ := timestamp.TimestampProto(time.Now())
	submission := pb.OutbreakEvent{LocationId: &locationID, StartTime: startTime, EndTime: endTime}

	mock.ExpectExec(
		`INSERT INTO qr_outbreak_events
		(location_id, originator, start_time, end_time, severity)
		VALUES (?, ?, ?, ?, ?)`).WithArgs(
		AnyType{},
		originator,
		AnyType{},
		AnyType{},
		AnyType{},
	).WillReturnResult(sqlmock.NewResult(1, 1))

	receivedError := conn.NewOutbreakEvent(context.TODO(), originator, &submission)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Nil(t, receivedError, "Expected nil if could execute insert")
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
		`INSERT INTO tek_upload_count
		(originator, date, count, first_upload)
		VALUES (?, ?, ?, ?)`,
	).WithArgs(
		"randomOrigin",
		time.Now().Format("2006-01-02"),
		len(keys),
		false,
	).WillReturnResult(sqlmock.NewResult(1, 1))

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
	receivedResult := conn.StoreKeys(pub, keys, nil)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Nil(t, receivedResult, "Expected nil when keys are commited")
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

	onsetDays := int32(0)
	expectedResult := []*pb.TemporaryExposureKey{
		&pb.TemporaryExposureKey{
			KeyData:                    []byte{},
			TransmissionRiskLevel:      &transmissionRiskLevel,
			RollingStartIntervalNumber: &currentRollingStartIntervalNumber,
			RollingPeriod:              &rollingPeriod,
			ReportType:                 pb.TemporaryExposureKey_CONFIRMED_TEST.Enum(),
			DaysSinceOnsetOfSymptoms:   &onsetDays,
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

func TestFetchOutbreakForTimeRange(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(allQueryMatcher))
	defer db.Close()

	conn := conn{
		db: db,
	}

	// No errors
	locationID := "ABCDEFGH"
	startTime, _ := timestamp.TimestampProto(time.Unix(1613238163, 0))
	endTime, _ := timestamp.TimestampProto(time.Unix(1613324563, 0))
	severity := uint32(1)
	submission := pb.OutbreakEvent{LocationId: &locationID, StartTime: startTime, EndTime: endTime, Severity: &severity}

	row := sqlmock.NewRows([]string{"location_id", "start_time", "end_time", "severity"}).AddRow(locationID, startTime.Seconds, endTime.Seconds, severity)
	mock.ExpectQuery("").WillReturnRows(row)

	expectedResult := []*pb.OutbreakEvent{&submission}

	receivedResult, _ := conn.FetchOutbreakForTimeRange(time.Now(), time.Now().Add(time.Hour*24))

	assert.Equal(t, expectedResult, receivedResult, "Expected rows for the query")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	// Errors
	mock.ExpectQuery("").WillReturnError(fmt.Errorf("Generic error"))

	_, receivedError := conn.FetchOutbreakForTimeRange(time.Now(), time.Now().Add(time.Hour*24))

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
