package persistence

import (
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/CovidShield/server/pkg/config"
	"github.com/CovidShield/server/pkg/timemath"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/nacl/box"
)

func TestDeleteOldDiagnosisKeys(t *testing.T) {
	// Init config
	config.InitConfig()

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

	// If query fails rollback transaction
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()

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
