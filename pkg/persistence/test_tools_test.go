package persistence

import (
	"fmt"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/cds-snc/covid-alert-server/pkg/testhelpers"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestTestTools_EnableTestToolsIsDisabled(t *testing.T) {
	assert.Panics(t, func() { clearDiagnosisKeys(nil,nil) }, "Should panic if ENABLE_TEST_TOOLS is not enabled")
}

func TestTestTools_EnableTestToolsIsFalse(t *testing.T) {
	os.Setenv("ENABLE_TEST_TOOLS", "false")
	assert.Panics(t, func() { clearDiagnosisKeys(nil,nil) }, "Should panic if ENABLE_TEST_TOOLS is set to false")
}

func TestTestTools_EnableTestToolsIsSetToSomethingOtherThanFalse(t *testing.T) {
	os.Setenv("ENABLE_TEST_TOOLS", "foo")
	assert.Panics(t, func() { clearDiagnosisKeys(nil,nil) }, "Should panic if ENABLE_TEST_TOOLS is not true")
}

func TestTestTools_EnableTestToolsIsEnabled(t *testing.T) {
	os.Setenv("ENABLE_TEST_TOOLS", "true")

	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()

	hook, oldLog := testhelpers.SetupTestLogging(&log)
	defer func () { log = *oldLog }()

	mock.ExpectExec(`TRUNCATE TABLE diagnosis_keys`).WillReturnResult(sqlmock.NewResult(0,0))

	assert.NotPanics(t, func() {_ = clearDiagnosisKeys(nil, db) }, "expect to not panic if ENABLE_TEST_TOOLS is true")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %v", err)
	}

	assertLog(t, hook, 1, logrus.InfoLevel, "diagnosis_keys was truncated")
}

func TestTestTools_TruncFailed(t *testing.T) {
	os.Setenv("ENABLE_TEST_TOOLS", "true")

	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()

	mock.ExpectExec(`TRUNCATE TABLE diagnosis_keys`).WillReturnError(fmt.Errorf("oh no"))

	err := clearDiagnosisKeys(nil, db)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %v", err)
	}

	assert.EqualError(t, err, "oh no", "should return an error if unable to truncate the table")
}

