package persistence

import (
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/cds-snc/covid-alert-server/pkg/config"
	pb "github.com/cds-snc/covid-alert-server/pkg/proto/covidshield"
	"github.com/cds-snc/covid-alert-server/pkg/timemath"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/nacl/box"
)

func TestDeleteOldDiagnosisKeys(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()

	oldestDateNumber := timemath.DateNumber(time.Now()) - config.AppConstants.MaxDiagnosisKeyRetentionDays
	oldestHour := timemath.HourNumberAtStartOfDate(oldestDateNumber)

	mock.ExpectExec(`DELETE FROM diagnosis_keys WHERE hour_of_submission < ?`).WithArgs(oldestHour).WillReturnResult(sqlmock.NewResult(1, 1))
	deleteOldDiagnosisKeys(db)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

}

func TestDeleteOldEncryptionKeys(t *testing.T) {

	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()

	query := fmt.Sprintf(`
		DELETE FROM encryption_keys
		WHERE  (created < (NOW() - INTERVAL %d DAY))
		OR    ((created < (NOW() - INTERVAL %d MINUTE)) AND app_public_key IS NULL)
		OR    remaining_keys = 0
	`, config.AppConstants.EncryptionKeyValidityDays, config.AppConstants.OneTimeCodeExpiryInMinutes)

	mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	deleteOldEncryptionKeys(db)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

}

func TestCaimKey(t *testing.T) {

	pub, _, _ := box.GenerateKey(rand.Reader)
	oneTimeCode := "80311300"

	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()

	// If query fails rollback transaction
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT COUNT(*) FROM encryption_keys WHERE app_public_key = ?`).WithArgs(pub[:]).WillReturnError(fmt.Errorf("error"))
	mock.ExpectRollback()
	_, receivedErr := claimKey(db, oneTimeCode, pub[:])

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr := fmt.Errorf("error")
	assert.Equal(t, expectedErr, receivedErr, "Expected error if could not query for key")

	// If app key exists
	mock.ExpectBegin()
	rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT COUNT(*) FROM encryption_keys WHERE app_public_key = ?`).WithArgs(pub[:]).WillReturnRows(rows)
	mock.ExpectRollback()
	_, receivedErr = claimKey(db, oneTimeCode, pub[:])

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr = ErrDuplicateKey
	assert.Equal(t, expectedErr, receivedErr, "Expected ErrDuplicateKey if there are duplicate keys")

	// App key does not exist, but created is not correct
	mock.ExpectBegin()
	rows = sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT COUNT(*) FROM encryption_keys WHERE app_public_key = ?`).WithArgs(pub[:]).WillReturnRows(rows)

	rows = sqlmock.NewRows([]string{"created"}).AddRow("1950-01-01 00:00:00")
	mock.ExpectQuery(`SELECT created FROM encryption_keys WHERE one_time_code = ?`).WithArgs(oneTimeCode).WillReturnRows(rows)

	mock.ExpectRollback()
	_, receivedErr = claimKey(db, oneTimeCode, pub[:])

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr = ErrInvalidOneTimeCode
	assert.Equal(t, expectedErr, receivedErr, "Expected ErrInvalidOneTimeCode if time code is not valid")

	// Prepare update fails
	mock.ExpectBegin()
	rows = sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT COUNT(*) FROM encryption_keys WHERE app_public_key = ?`).WithArgs(pub[:]).WillReturnRows(rows)

	rows = sqlmock.NewRows([]string{"created"}).AddRow(time.Now())
	mock.ExpectQuery(`SELECT created FROM encryption_keys WHERE one_time_code = ?`).WithArgs(oneTimeCode).WillReturnRows(rows)

	query := fmt.Sprintf(
		`UPDATE encryption_keys
		SET one_time_code = NULL,
			app_public_key = ?,
			created = ?
		WHERE one_time_code = ?
		AND created > (NOW() - INTERVAL %d MINUTE)`,
		config.AppConstants.OneTimeCodeExpiryInMinutes,
	)

	mock.ExpectPrepare(query).WillReturnError(fmt.Errorf("error"))

	mock.ExpectRollback()
	_, receivedErr = claimKey(db, oneTimeCode, pub[:])

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr = fmt.Errorf("error")
	assert.Equal(t, expectedErr, receivedErr, "Expected error if could not prepare update")

	// Execute fails after update
	mock.ExpectBegin()
	rows = sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT COUNT(*) FROM encryption_keys WHERE app_public_key = ?`).WithArgs(pub[:]).WillReturnRows(rows)

	created := time.Now()

	rows = sqlmock.NewRows([]string{"created"}).AddRow(created)
	mock.ExpectQuery(`SELECT created FROM encryption_keys WHERE one_time_code = ?`).WithArgs(oneTimeCode).WillReturnRows(rows)

	created = timemath.MostRecentUTCMidnight(created)

	mock.ExpectPrepare(query).ExpectExec().WithArgs(pub[:], created, oneTimeCode).WillReturnError(fmt.Errorf("error"))

	mock.ExpectRollback()
	_, receivedErr = claimKey(db, oneTimeCode, pub[:])

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr = fmt.Errorf("error")
	assert.Equal(t, expectedErr, receivedErr, "Expected error if could not execute update")

	// RowsAffected is not equal to 1
	mock.ExpectBegin()
	rows = sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT COUNT(*) FROM encryption_keys WHERE app_public_key = ?`).WithArgs(pub[:]).WillReturnRows(rows)

	created = time.Now()

	rows = sqlmock.NewRows([]string{"created"}).AddRow(created)
	mock.ExpectQuery(`SELECT created FROM encryption_keys WHERE one_time_code = ?`).WithArgs(oneTimeCode).WillReturnRows(rows)

	created = timemath.MostRecentUTCMidnight(created)

	mock.ExpectPrepare(query).ExpectExec().WithArgs(pub[:], created, oneTimeCode).WillReturnResult(sqlmock.NewResult(1, 2))

	mock.ExpectRollback()
	_, receivedErr = claimKey(db, oneTimeCode, pub[:])

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr = ErrInvalidOneTimeCode
	assert.Equal(t, expectedErr, receivedErr, "Expected ErrInvalidOneTimeCode if rowsAffected was not 1")

	// Getting public key throws an error
	mock.ExpectBegin()
	rows = sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT COUNT(*) FROM encryption_keys WHERE app_public_key = ?`).WithArgs(pub[:]).WillReturnRows(rows)

	created = time.Now()

	rows = sqlmock.NewRows([]string{"created"}).AddRow(created)
	mock.ExpectQuery(`SELECT created FROM encryption_keys WHERE one_time_code = ?`).WithArgs(oneTimeCode).WillReturnRows(rows)

	created = timemath.MostRecentUTCMidnight(created)

	mock.ExpectPrepare(query).ExpectExec().WithArgs(pub[:], created, oneTimeCode).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectPrepare(`SELECT server_public_key FROM encryption_keys WHERE app_public_key = ?`).ExpectQuery().WithArgs(pub[:]).WillReturnError(fmt.Errorf("error"))

	mock.ExpectRollback()
	_, receivedErr = claimKey(db, oneTimeCode, pub[:])

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr = fmt.Errorf("error")
	assert.Equal(t, expectedErr, receivedErr, "Expected error if server_public_key was not queried")

	// Commits and returns a server key
	mock.ExpectBegin()
	rows = sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT COUNT(*) FROM encryption_keys WHERE app_public_key = ?`).WithArgs(pub[:]).WillReturnRows(rows)

	created = time.Now()

	rows = sqlmock.NewRows([]string{"created"}).AddRow(created)
	mock.ExpectQuery(`SELECT created FROM encryption_keys WHERE one_time_code = ?`).WithArgs(oneTimeCode).WillReturnRows(rows)

	created = timemath.MostRecentUTCMidnight(created)

	mock.ExpectPrepare(query).ExpectExec().WithArgs(pub[:], created, oneTimeCode).WillReturnResult(sqlmock.NewResult(1, 1))

	rows = sqlmock.NewRows([]string{"server_public_key"}).AddRow(pub[:])
	mock.ExpectPrepare(`SELECT server_public_key FROM encryption_keys WHERE app_public_key = ?`).ExpectQuery().WithArgs(pub[:]).WillReturnRows(rows)

	mock.ExpectCommit()

	serverKey, _ := claimKey(db, oneTimeCode, pub[:])

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, pub[:], serverKey, "should return server key")

}

func TestPersistEncryptionKey(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()

	region := "302"
	originator := "randomOrigin"
	pub, priv, _ := box.GenerateKey(rand.Reader)
	oneTimeCode := "80311300"

	// Return error
	mock.ExpectExec(
		`INSERT INTO encryption_keys
		(region, originator, server_private_key, server_public_key, one_time_code, remaining_keys)
		VALUES (?, ?, ?, ?, ?, ?)`).WithArgs(
		region,
		originator,
		priv[:],
		pub[:],
		oneTimeCode,
		config.AppConstants.InitialRemainingKeys,
	).WillReturnError(fmt.Errorf("error"))

	receivedErr := persistEncryptionKey(db, region, originator, pub, priv, oneTimeCode)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr := fmt.Errorf("error")
	assert.Equal(t, expectedErr, receivedErr, "Expected error if could not execute insert")

	// Success
	mock.ExpectExec(
		`INSERT INTO encryption_keys
		(region, originator, server_private_key, server_public_key, one_time_code, remaining_keys)
		VALUES (?, ?, ?, ?, ?, ?)`).WithArgs(
		region,
		originator,
		priv[:],
		pub[:],
		oneTimeCode,
		config.AppConstants.InitialRemainingKeys,
	).WillReturnResult(sqlmock.NewResult(1, 1))

	receivedResult := persistEncryptionKey(db, region, originator, pub, priv, oneTimeCode)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Nil(t, receivedResult, "Expected nil if it could execute insert")

}

func testPersistEncryptionKeyWithHashID(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()

	region := "302"
	originator := "randomOrigin"
	pub, priv, _ := box.GenerateKey(rand.Reader)
	oneTimeCode := "80311300"
	hashID := "abcd"

	// Return error if unknown error
	mock.ExpectExec(
		`INSERT INTO encryption_keys
		(region, originator, server_private_key, server_public_key, one_time_code, remaining_keys)
		VALUES (?, ?, ?, ?, ?, ?)`).WithArgs(
		region,
		originator,
		hashID,
		priv[:],
		pub[:],
		oneTimeCode,
		config.AppConstants.InitialRemainingKeys,
	).WillReturnError(fmt.Errorf("error"))

	receivedErr := persistEncryptionKeyWithHashID(db, region, originator, hashID, pub, priv, oneTimeCode)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr := fmt.Errorf("error")
	assert.Equal(t, expectedErr, receivedErr, "Expected error if could not execute update")

	// Return error if duplicate one_time_code
	mock.ExpectExec(
		`INSERT INTO encryption_keys
		(region, originator, server_private_key, server_public_key, one_time_code, remaining_keys)
		VALUES (?, ?, ?, ?, ?, ?)`).WithArgs(
		region,
		originator,
		hashID,
		priv[:],
		pub[:],
		oneTimeCode,
		config.AppConstants.InitialRemainingKeys,
	).WillReturnError(fmt.Errorf("for key 'one_time_code"))

	receivedErr = persistEncryptionKeyWithHashID(db, region, originator, hashID, pub, priv, oneTimeCode)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr = fmt.Errorf("for key 'one_time_code")
	assert.Equal(t, expectedErr, receivedErr, "Expected error if could not execute insert")

	// Return error if duplicate used hashID found
	mock.ExpectExec(
		`INSERT INTO encryption_keys
		(region, originator, server_private_key, server_public_key, one_time_code, remaining_keys)
		VALUES (?, ?, ?, ?, ?, ?)`).WithArgs(
		region,
		originator,
		hashID,
		priv[:],
		pub[:],
		oneTimeCode,
		config.AppConstants.InitialRemainingKeys,
	).WillReturnError(fmt.Errorf("for key 'hash_id"))

	rows := sqlmock.NewRows([]string{"one_time_code"}).AddRow(nil)
	mock.ExpectQuery(
		`SELECT one_time_code FROM encryption_keys WHERE hash_id = ? FOR UPDATE`).WithArgs(hashID).WillReturnRows(rows)

	receivedErr = persistEncryptionKeyWithHashID(db, region, originator, hashID, pub, priv, oneTimeCode)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr = fmt.Errorf("used hashID found")
	assert.Equal(t, expectedErr, receivedErr, "Expected error if could not execute insert")

	// Return error if duplicate un-used hashID found but delete fails
	mock.ExpectExec(
		`INSERT INTO encryption_keys
		(region, originator, server_private_key, server_public_key, one_time_code, remaining_keys)
		VALUES (?, ?, ?, ?, ?, ?)`).WithArgs(
		region,
		originator,
		hashID,
		priv[:],
		pub[:],
		oneTimeCode,
		config.AppConstants.InitialRemainingKeys,
	).WillReturnError(fmt.Errorf("for key 'for key 'hash_id"))

	rows = sqlmock.NewRows([]string{"one_time_code"}).AddRow(oneTimeCode)
	mock.ExpectQuery(
		`SELECT one_time_code FROM encryption_keys WHERE hash_id = ? FOR UPDATE`).WithArgs(hashID).WillReturnRows(rows)
	mock.ExpectExec(`DELETE FROM encryption_keys WHERE hash_id = ? AND one_time_code IS NOT NULL`).WithArgs(hashID).WillReturnError(fmt.Errorf("error"))

	receivedErr = persistEncryptionKeyWithHashID(db, region, originator, hashID, pub, priv, oneTimeCode)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr = fmt.Errorf("error")
	assert.Equal(t, expectedErr, receivedErr, "Expected error if could not execute delete")

	// Return error if duplicate un-used hashID found and delete passes (regenerates OTC)
	mock.ExpectExec(
		`INSERT INTO encryption_keys
		(region, originator, server_private_key, server_public_key, one_time_code, remaining_keys)
		VALUES (?, ?, ?, ?, ?, ?)`).WithArgs(
		region,
		originator,
		hashID,
		priv[:],
		pub[:],
		oneTimeCode,
		config.AppConstants.InitialRemainingKeys,
	).WillReturnError(fmt.Errorf("for key 'for key 'hash_id"))

	rows = sqlmock.NewRows([]string{"one_time_code"}).AddRow(oneTimeCode)
	mock.ExpectQuery(
		`SELECT one_time_code FROM encryption_keys WHERE hash_id = ? FOR UPDATE`).WithArgs(hashID).WillReturnRows(rows)
	mock.ExpectExec(`DELETE FROM encryption_keys WHERE hash_id = ? AND one_time_code IS NOT NULL`).WithArgs(hashID).WillReturnResult(sqlmock.NewResult(1, 1))

	receivedErr = persistEncryptionKeyWithHashID(db, region, originator, hashID, pub, priv, oneTimeCode)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr = fmt.Errorf("regenerate OTC for hashID")
	assert.Equal(t, expectedErr, receivedErr, "Expected error if could execute delete")

	// Success
	mock.ExpectExec(
		`INSERT INTO encryption_keys
		(region, originator, server_private_key, server_public_key, one_time_code, remaining_keys)
		VALUES (?, ?, ?, ?, ?, ?)`).WithArgs(
		region,
		originator,
		hashID,
		priv[:],
		pub[:],
		oneTimeCode,
		config.AppConstants.InitialRemainingKeys,
	).WillReturnResult(sqlmock.NewResult(1, 1))

	receivedResult := persistEncryptionKeyWithHashID(db, region, originator, hashID, pub, priv, oneTimeCode)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Nil(t, receivedResult, "Expected nothing if could execute insert")

}

func TestPrivForPub(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()

	pub, priv, _ := box.GenerateKey(rand.Reader)

	query := fmt.Sprintf(`
	SELECT server_private_key FROM encryption_keys
		WHERE server_public_key = ?
		AND created > (NOW() - INTERVAL %d DAY)
		LIMIT 1`,
		config.AppConstants.EncryptionKeyValidityDays,
	)

	rows := sqlmock.NewRows([]string{"server_private_key"}).AddRow(priv[:])
	mock.ExpectQuery(query).WithArgs(pub[:]).WillReturnRows(rows)

	expectedResult := priv[:]
	var receivedResult []byte
	privForPub(db, pub[:]).Scan(&receivedResult)

	assert.Equal(t, expectedResult, receivedResult, "Expected private key for public key")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestDiagnosisKeysForHours(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()

	region := "302"
	startHour := uint32(100)
	endHour := uint32(200)
	currentRollingStartIntervalNumber := int32(2651450)
	minRollingStartIntervalNumber := timemath.RollingStartIntervalNumberPlusDays(currentRollingStartIntervalNumber, -14)

	query := `
	SELECT region, key_data, rolling_start_interval_number, rolling_period, transmission_risk_level FROM diagnosis_keys
		WHERE hour_of_submission >= ?
		AND hour_of_submission < ?
		AND rolling_start_interval_number > ?
		AND region = ?
		ORDER BY key_data`

	row := sqlmock.NewRows([]string{"region", "key_data", "rolling_start_interval_number", "rolling_period", "transmission_risk_level"}).AddRow("302", []byte{}, 2651450, 144, 4)
	mock.ExpectQuery(query).WithArgs(
		startHour,
		endHour,
		minRollingStartIntervalNumber,
		region).WillReturnRows(row)

	expectedResult := []byte("302")
	rows, _ := diagnosisKeysForHours(db, region, startHour, endHour, currentRollingStartIntervalNumber)
	var receivedResult []byte
	for rows.Next() {
		rows.Scan(&receivedResult, nil, nil, nil, nil)
	}

	assert.Equal(t, expectedResult, receivedResult, "Expected rows for the query")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRegisterDiagnosisKeys(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()

	pub, _, _ := box.GenerateKey(rand.Reader)
	keys := []*pb.TemporaryExposureKey{}
	region := "302"
	originator := "randomOrigin"

	// Roll back if table is locked
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT region, originator, remaining_keys FROM encryption_keys WHERE app_public_key = ? FOR UPDATE`).WithArgs(pub[:]).WillReturnError(fmt.Errorf("error"))
	mock.ExpectRollback()
	receivedErr := registerDiagnosisKeys(db, pub, keys)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr := fmt.Errorf("error")
	assert.Equal(t, expectedErr, receivedErr, "Expected error if table is locked")

	// Roll back if 0 keys are left and return error
	mock.ExpectBegin()
	row := sqlmock.NewRows([]string{"region", "originator", "remaining_keys"}).AddRow(region, originator, 0)
	mock.ExpectQuery(`SELECT region, originator, remaining_keys FROM encryption_keys WHERE app_public_key = ? FOR UPDATE`).WillReturnRows(row)
	mock.ExpectRollback()
	receivedErr = registerDiagnosisKeys(db, pub, keys)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr = ErrKeyConsumed
	assert.Equal(t, expectedErr, receivedErr, "Expected ErrKeyConsumed if 0 keys are left")

	// Roll back if prepare fails
	mock.ExpectBegin()
	row = sqlmock.NewRows([]string{"region", "originator", "remaining_keys"}).AddRow("302", "randomOrigin", 1)
	mock.ExpectQuery(`SELECT region, originator, remaining_keys FROM encryption_keys WHERE app_public_key = ? FOR UPDATE`).WillReturnRows(row)
	mock.ExpectPrepare(
		`INSERT IGNORE INTO diagnosis_keys
		(region, originator, key_data, rolling_start_interval_number, rolling_period, transmission_risk_level, hour_of_submission)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
	).WillReturnError(fmt.Errorf("error"))

	mock.ExpectRollback()
	receivedErr = registerDiagnosisKeys(db, pub, keys)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr = fmt.Errorf("error")
	assert.Equal(t, expectedErr, receivedErr, "Expected error if prepare to insert fails")

	// Rolls back if it fails to execute insertion of keys
	key := randomTestKey()
	keys = []*pb.TemporaryExposureKey{key}

	hourOfSubmission := timemath.HourNumber(time.Now())

	mock.ExpectBegin()
	row = sqlmock.NewRows([]string{"region", "originator", "remaining_keys"}).AddRow("302", "randomOrigin", 1)
	mock.ExpectQuery(`SELECT region, originator, remaining_keys FROM encryption_keys WHERE app_public_key = ? FOR UPDATE`).WillReturnRows(row)
	mock.ExpectPrepare(
		`INSERT IGNORE INTO diagnosis_keys
		(region, originator, key_data, rolling_start_interval_number, rolling_period, transmission_risk_level, hour_of_submission)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
	).ExpectExec().WithArgs(
		region,
		originator,
		key.GetKeyData(),
		key.GetRollingStartIntervalNumber(),
		key.GetRollingPeriod(),
		key.GetTransmissionRiskLevel(),
		hourOfSubmission,
	).WillReturnError(fmt.Errorf("error"))

	mock.ExpectRollback()
	receivedErr = registerDiagnosisKeys(db, pub, keys)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr = fmt.Errorf("error")
	assert.Equal(t, expectedErr, receivedErr, "Expected error if execute insert fails")

	// Rolls back if more keys are inserted that are allowed
	keyOne := randomTestKey()
	keyTwo := randomTestKey()
	keys = []*pb.TemporaryExposureKey{keyOne, keyTwo}

	hourOfSubmission = timemath.HourNumber(time.Now())

	mock.ExpectBegin()
	row = sqlmock.NewRows([]string{"region", "originator", "remaining_keys"}).AddRow("302", "randomOrigin", 1)
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

	mock.ExpectRollback()
	receivedErr = registerDiagnosisKeys(db, pub, keys)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr = ErrTooManyKeys
	assert.Equal(t, expectedErr, receivedErr, "Expected error more keys than allowed are inserted")

	// Rolls back if final update fails
	keyOne = randomTestKey()
	keyTwo = randomTestKey()
	keys = []*pb.TemporaryExposureKey{keyOne, keyTwo}

	hourOfSubmission = timemath.HourNumber(time.Now())

	mock.ExpectBegin()
	row = sqlmock.NewRows([]string{"region", "originator", "remaining_keys"}).AddRow("302", "randomOrigin", 3)
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
	).WillReturnError(fmt.Errorf("error"))

	mock.ExpectRollback()
	receivedErr = registerDiagnosisKeys(db, pub, keys)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedErr = ErrTooManyKeys
	assert.Equal(t, expectedErr, receivedErr, "Expected error more keys than allowed are inserted")

	// Commits and logs how many keys were inserted
	keyOne = randomTestKey()
	keyTwo = randomTestKey()
	keys = []*pb.TemporaryExposureKey{keyOne, keyTwo}

	hourOfSubmission = timemath.HourNumber(time.Now())

	mock.ExpectBegin()
	row = sqlmock.NewRows([]string{"region", "originator", "remaining_keys"}).AddRow("302", "randomOrigin", 3)
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
	receivedResult := registerDiagnosisKeys(db, pub, keys)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Nil(t, receivedResult, "Expected nil when keys are commited")
}

func TestCheckClaimKeyBan(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()

	var maxConsecutiveClaimKeyFailures = config.AppConstants.MaxConsecutiveClaimKeyFailures

	identifier := "127.0.0.1"

	// Queries and succeeds if no result is found
	row := sqlmock.NewRows([]string{"failures", "last_failure"})
	mock.ExpectQuery(`SELECT failures, last_failure FROM failed_key_claim_attempts WHERE identifier = ?`).WithArgs(identifier).WillReturnRows(row)

	expectedTriesRemaining := maxConsecutiveClaimKeyFailures
	expectedBanDuration := time.Duration(0)

	receivedTriesRemaining, receivedBanDuration, _ := checkClaimKeyBan(db, identifier)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, expectedTriesRemaining, receivedTriesRemaining, "Expected maxConsecutiveClaimKeyFailures as tries remaining")
	assert.Equal(t, expectedBanDuration, receivedBanDuration, "Expected 0 as ban duration")

	// Queries and fails if an unkown error is returned
	mock.ExpectQuery(`SELECT failures, last_failure FROM failed_key_claim_attempts WHERE identifier = ?`).WithArgs(identifier).WillReturnError(fmt.Errorf("error"))

	expectedTriesRemaining = 0
	expectedBanDuration = time.Duration(0)
	expectedErr := fmt.Errorf("error")

	receivedTriesRemaining, receivedBanDuration, receivedErr := checkClaimKeyBan(db, identifier)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, expectedTriesRemaining, receivedTriesRemaining, "Expected maxConsecutiveClaimKeyFailures as tries remaining")
	assert.Equal(t, expectedBanDuration, receivedBanDuration, "Expected 0 as ban duration")
	assert.Equal(t, expectedErr, receivedErr, "Expected error if unkown error occures")

	// Returns correct tries remaining if not banned
	attempts := 1
	row = sqlmock.NewRows([]string{"failures", "last_failure"}).AddRow(attempts, time.Now())
	mock.ExpectQuery(`SELECT failures, last_failure FROM failed_key_claim_attempts WHERE identifier = ?`).WithArgs(identifier).WillReturnRows(row)

	expectedTriesRemaining = maxConsecutiveClaimKeyFailures - attempts
	expectedBanDuration = time.Duration(0)

	receivedTriesRemaining, receivedBanDuration, _ = checkClaimKeyBan(db, identifier)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, expectedTriesRemaining, receivedTriesRemaining, "Expected maxConsecutiveClaimKeyFailures - attempts as tries remaining")
	assert.Equal(t, expectedBanDuration, receivedBanDuration, "Expected 0 as ban duration")

	// Returns correct banDuration if banned
	attempts = maxConsecutiveClaimKeyFailures
	row = sqlmock.NewRows([]string{"failures", "last_failure"}).AddRow(attempts, time.Now())
	mock.ExpectQuery(`SELECT failures, last_failure FROM failed_key_claim_attempts WHERE identifier = ?`).WithArgs(identifier).WillReturnRows(row)

	expectedTriesRemaining = maxConsecutiveClaimKeyFailures - attempts
	expectedBanDuration, _ = time.ParseDuration("59m59s")

	receivedTriesRemaining, receivedBanDuration, _ = checkClaimKeyBan(db, identifier)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, expectedTriesRemaining, receivedTriesRemaining, "Expected 0 as tries remaining")
	assert.GreaterOrEqual(t, receivedBanDuration.Seconds(), expectedBanDuration.Seconds(), "Expected something greater than 59m59s as ban duration")

	// Resets if banDuration has expired
	attempts = maxConsecutiveClaimKeyFailures
	row = sqlmock.NewRows([]string{"failures", "last_failure"}).AddRow(attempts, time.Now().Add(-time.Hour*1))
	mock.ExpectQuery(`SELECT failures, last_failure FROM failed_key_claim_attempts WHERE identifier = ?`).WithArgs(identifier).WillReturnRows(row)

	expectedTriesRemaining = maxConsecutiveClaimKeyFailures
	expectedBanDuration = time.Duration(0)

	receivedTriesRemaining, receivedBanDuration, _ = checkClaimKeyBan(db, identifier)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, expectedTriesRemaining, receivedTriesRemaining, "Expected maxConsecutiveClaimKeyFailures as tries remaining")
	assert.Equal(t, expectedBanDuration, receivedBanDuration, "Expected 0 as ban duration")
}

func TestRegisterClaimKeySuccess(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()

	identifier := "127.0.0.1"

	mock.ExpectExec(`DELETE FROM failed_key_claim_attempts WHERE identifier = ?`).WithArgs(identifier).WillReturnResult(sqlmock.NewResult(1, 1))
	receivedResult := registerClaimKeySuccess(db, identifier)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Nil(t, receivedResult, "Expected nil if executed delete")
}

func TestRegisterClaimKeyFailure(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()

	identifier := "127.0.0.1"
	var maxConsecutiveClaimKeyFailures = config.AppConstants.MaxConsecutiveClaimKeyFailures

	// Roll back if insert fails
	mock.ExpectBegin()
	mock.ExpectExec(
		`INSERT INTO failed_key_claim_attempts (identifier) VALUES (?)
		ON DUPLICATE KEY UPDATE
      failures = failures + 1,
			last_failure = NOW()`).WithArgs(
		identifier,
	).WillReturnError(fmt.Errorf("error"))
	mock.ExpectRollback()

	expectedTriesRemaining := 0
	expectedBanDuration := time.Duration(0)
	expectedErr := fmt.Errorf("error")

	receivedTriesRemaining, receivedBanDuration, receivedErr := registerClaimKeyFailure(db, identifier)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, expectedTriesRemaining, receivedTriesRemaining, "Expected maxConsecutiveClaimKeyFailures as tries remaining")
	assert.Equal(t, expectedBanDuration, receivedBanDuration, "Expected 0 as ban duration")
	assert.Equal(t, expectedErr, receivedErr, "Expected error if could not execute update")

	// Rolls back if checkClaimKeyBan returns an error
	mock.ExpectBegin()
	mock.ExpectExec(
		`INSERT INTO failed_key_claim_attempts (identifier) VALUES (?)
		ON DUPLICATE KEY UPDATE
      failures = failures + 1,
			last_failure = NOW()`).WithArgs(
		identifier,
	).WillReturnResult(sqlmock.NewResult(1, 1))

	//--> Called in checkClaimKeyBan
	mock.ExpectQuery(`SELECT failures, last_failure FROM failed_key_claim_attempts WHERE identifier = ?`).WithArgs(identifier).WillReturnError(fmt.Errorf("error"))

	mock.ExpectRollback()

	expectedTriesRemaining = 0
	expectedBanDuration = time.Duration(0)
	expectedErr = fmt.Errorf("error")

	receivedTriesRemaining, receivedBanDuration, receivedErr = registerClaimKeyFailure(db, identifier)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, expectedTriesRemaining, receivedTriesRemaining, "Expected maxConsecutiveClaimKeyFailures as tries remaining")
	assert.Equal(t, expectedBanDuration, receivedBanDuration, "Expected 0 as ban duration")
	assert.Equal(t, expectedErr, receivedErr, "Expected error if could not execute update")

	// Commits and returns the correct data from checkClaimKeyBan

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

	expectedTriesRemaining = maxConsecutiveClaimKeyFailures - 1
	expectedBanDuration = time.Duration(0)

	receivedTriesRemaining, receivedBanDuration, receivedErr = registerClaimKeyFailure(db, identifier)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, expectedTriesRemaining, receivedTriesRemaining, "Expected maxConsecutiveClaimKeyFailures - 1 as tries remaining")
	assert.Equal(t, expectedBanDuration, receivedBanDuration, "Expected 0 as ban duration")
	assert.Nil(t, receivedErr, "Expected no error if inserted")
}

func TestDeleteOldFailedClaimKeyAttempts(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()

	mock.ExpectExec(`DELETE FROM failed_key_claim_attempts WHERE last_failure < ?`).WillReturnResult(sqlmock.NewResult(1, 1))

	expectedResult := int64(1)
	receivedResult, receivedError := deleteOldFailedClaimKeyAttempts(db)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, expectedResult, receivedResult, "Expected to only affect one row")
	assert.Nil(t, receivedError, "Expected nil if executed delete")
}

func TestCountClaimedOneTimeCodes(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()

	row := sqlmock.NewRows([]string{"count"}).AddRow(100)
	mock.ExpectQuery(`SELECT COUNT(*) FROM encryption_keys WHERE one_time_code IS NULL`).WillReturnRows(row)

	expectedResult := int64(100)

	receivedResult, receivedErr := countClaimedOneTimeCodes(db)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, expectedResult, receivedResult, "Expected to receive count of 100")
	assert.Nil(t, receivedErr, "Expected nil if query ran")
}

func TestCountDiagnosisKeys(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()

	row := sqlmock.NewRows([]string{"count"}).AddRow(100)
	mock.ExpectQuery(`SELECT COUNT(*) FROM diagnosis_keys`).WillReturnRows(row)

	expectedResult := int64(100)

	receivedResult, receivedErr := countDiagnosisKeys(db)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, expectedResult, receivedResult, "Expected to receive count of 100")
	assert.Nil(t, receivedErr, "Expected nil if query ran")
}

func TestCountUnclaimedOneTimeCodes(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()

	row := sqlmock.NewRows([]string{"count"}).AddRow(100)
	mock.ExpectQuery(`SELECT COUNT(*) FROM encryption_keys WHERE one_time_code IS NOT NULL`).WillReturnRows(row)

	expectedResult := int64(100)

	receivedResult, receivedErr := countUnclaimedOneTimeCodes(db)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, expectedResult, receivedResult, "Expected to receive count of 100")
	assert.Nil(t, receivedErr, "Expected nil if query ran")
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
