package persistence

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRunMigration(t *testing.T) {

	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))

	defer db.Close()

	for _, migration := range migrations {
		mock.ExpectBegin()
		rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
		mock.ExpectQuery(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`).WithArgs(migration.id).WillReturnRows(rows)
		for _, statement := range migration.statements {
			mock.ExpectExec(statement).WillReturnResult(sqlmock.NewResult(1, 1))
		}
		mock.ExpectExec("INSERT INTO schema_migrations (version) VALUES (?)").WithArgs(migration.id).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()
		runMigration(db, migration)
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	}

}
