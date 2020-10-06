package persistence

import (
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_countByOriginatorCallsQuery(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))

	rows := sqlmock.NewRows([]string{"originator", "count"}).AddRow("foo", 1).AddRow("bar",2)

	query := `select * from events`
	mock.ExpectQuery(query).WillReturnRows(rows)

	counts, _ := countByOriginator(db, query)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t,counts,[]CountByOriginator{{"foo",1}, {"bar", 2}})
}

func Test_countByOriginatorReturnsError(t *testing.T){

	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))

	query := `select * from events`
	mock.ExpectQuery(query).WillReturnError(fmt.Errorf("foo"))

	_, err := countByOriginator(db, query)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t,err,fmt.Errorf("foo"))
}



