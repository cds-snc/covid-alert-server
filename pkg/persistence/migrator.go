package persistence

import (
	"database/sql"
	"fmt"
	"strings"

	// inject mysql support for database/sql
	_ "github.com/go-sql-driver/mysql"
)

const ensureSchemaMigrations = `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) NOT NULL,
		UNIQUE KEY unique_schema_migrations (version)
	)`

type migration struct {
	id         string
	statements []string
}

var migrations = []migration{
	{
		id: "1",
		statements: []string{`
CREATE TABLE IF NOT EXISTS diagnosis_keys (
	key_data                      BINARY(16) NOT NULL UNIQUE,
	rolling_start_interval_number INT NOT NULL,               -- int32
	rolling_period                INT NOT NULL,               -- int32
	transmission_risk_level       SMALLINT UNSIGNED NOT NULL, -- uint16
	hour_of_submission            INT UNSIGNED NOT NULL,      -- uint32
	region                        VARCHAR(32) NOT NULL,

	UNIQUE INDEX (key_data, region),
	INDEX (key_data),
	INDEX (region),
	INDEX (hour_of_submission),
	INDEX (rolling_start_interval_number)
)`, `
CREATE TABLE IF NOT EXISTS encryption_keys (
	server_private_key   BINARY(32) NOT NULL UNIQUE,
	server_public_key    BINARY(32) NOT NULL UNIQUE,
	app_public_key       BINARY(32)          UNIQUE,
	one_time_code        VARCHAR(8)          UNIQUE,
	remaining_keys       SMALLINT UNSIGNED NOT NULL,
	created              TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	region               VARCHAR(32) NOT NULL,

	INDEX (one_time_code),
	INDEX (app_public_key, region),
	INDEX (region),
	INDEX (created)
)`,
		},
	}, {
		id: "2",
		statements: []string{`
CREATE TABLE IF NOT EXISTS failed_key_claim_attempts (
	identifier      VARCHAR(32)       NOT NULL UNIQUE,
	failures        SMALLINT UNSIGNED NOT NULL DEFAULT 1,
	last_failure    TIMESTAMP         NOT NULL DEFAULT CURRENT_TIMESTAMP,

	UNIQUE INDEX (identifier),
	INDEX (last_failure)
)`,
		},
	}, {
		id: "3",
		statements: []string{
			`ALTER TABLE diagnosis_keys  ADD COLUMN originator VARCHAR(64)`,
			`ALTER TABLE encryption_keys ADD COLUMN originator VARCHAR(64)`,
			`ALTER TABLE diagnosis_keys  ADD INDEX (originator)`,
			`ALTER TABLE encryption_keys ADD INDEX (originator)`,
		},
	},
}

// MigrateDatabase creates the database and migrates it into the correct state.
func MigrateDatabase(url string) error {
	parts := strings.Split(url, "/")
	dbName := parts[len(parts)-1]

	dbForCreate, err := sql.Open("mysql", strings.TrimSuffix(url, dbName))
	if err != nil {
		return err
	}

	if _, err := dbForCreate.Exec(`CREATE DATABASE IF NOT EXISTS ` + dbName); err != nil {
		return err
	}
	if err := dbForCreate.Close(); err != nil {
		return err
	}

	db, err := sql.Open("mysql", url)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log(nil, err).Error("migrator db close failed")
		}
	}()

	if _, err := db.Exec(ensureSchemaMigrations); err != nil {
		return err
	}

	// Multiple processes coming up and trying to run migrations at the same time
	// could be bad news if our migrations aren't idempotent and we don't lock.
	log(nil, nil).Info("locking table schema_migrations")
	if _, err := db.Exec("LOCK TABLES schema_migrations WRITE"); err != nil {
		return err
	}
	log(nil, nil).Info("acquired table lock on schema_migrations")

	for _, migration := range migrations {
		if err := runMigration(db, migration); err != nil {
			return err
		}
	}

	if _, err := db.Exec("UNLOCK TABLES"); err != nil {
		return err
	}
	log(nil, nil).Info("released table lock on schema_migrations")

	log(nil, nil).Info("migrations done")

	return nil
}

func runMigration(db *sql.DB, migration migration) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", migration.id).Scan(&count); err != nil {
		if err := tx.Rollback(); err != nil {
			return err
		}
		return err
	}
	if count == 1 {
		return tx.Rollback()
	}

	fmt.Printf("===== running migration id=%s ===============================\n", migration.id)
	for _, statement := range migration.statements {
		fmt.Println(statement)
		if _, err := db.Exec(statement); err != nil {
			if err := tx.Rollback(); err != nil {
				return err
			}
			return err
		}
	}

	if _, err := db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", migration.id); err != nil {
		if err := tx.Rollback(); err != nil {
			return err
		}
		return err
	}

	return tx.Commit()
}
