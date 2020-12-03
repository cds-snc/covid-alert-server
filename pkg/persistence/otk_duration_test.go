package persistence

import (
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
	"time"
)

func setupOtkDurationMock(mock sqlmock.Sqlmock, originator string, duration int) {
	mock.ExpectExec(`
		INSERT INTO otk_life_duration
		(originator, hours, date, count)
		VALUES(?, ?, ?, 1) ON DUPLICATE KEY UPDATE count = count + 1`).
		WithArgs( originator, duration, AnyType{}).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

func Test_roundToNearestHourLessThan1Hour(t *testing.T){
	db, mock := createNewSqlMock()
	defer db.Close()

	setupOtkDurationMock(mock, "foo", 1)

	d, _ := time.ParseDuration("30m")
	od := OtkDuration{
		"foo",
		d,
	}

	saveOtkDuration(db, od)
}

func Test_roundToNearestHourSixHours(t *testing.T){
	db, mock := createNewSqlMock()
	defer db.Close()

	setupOtkDurationMock(mock, "foo", 1)

	d, _ := time.ParseDuration("5h15m")
	od := OtkDuration{
		"foo",
		d,
	}

	saveOtkDuration(db, od)
}
