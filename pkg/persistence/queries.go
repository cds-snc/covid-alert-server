package persistence

import (
	"database/sql"
	"fmt"
	"time"

	pb "github.com/CovidShield/server/pkg/proto/covidshield"
	"github.com/CovidShield/server/pkg/timemath"
)

// (Legal requirement: <21)
// We serve up the last 14. This number 15 includes the current day, so 14 days
// ago is the oldest data.
const maxDiagnosisKeyRetentionDays = 15

// A generated keypair can upload up to 28 keys (14 on day 1, plus 14
// subsequent days)
const initialRemainingKeys = 28

// (Legal requirement: <21)
// When we assign an Application Public Key to a server keypair, we reset the
// created timestamp to the beginning of its existing UTC date. (i.e.
// subtracting anywhere from 00:00 to 23:59 from it)
//
// From that timestamp, the Application may submit keys for up to 15 days,
// which really means they should submit keys for up to 14 days.
const encryptionKeyValidityDays = 15

// OneTimeCodes must be used within 10 minutes, otherwise they expire.
const oneTimeCodeExpiryInMinutes = 10

func deleteOldDiagnosisKeys(db *sql.DB) (int64, error) {
	oldestDateNumber := timemath.DateNumber(time.Now()) - maxDiagnosisKeyRetentionDays
	oldestHour := timemath.HourNumberAtStartOfDate(oldestDateNumber)

	res, err := db.Exec(`DELETE FROM diagnosis_keys WHERE hour_of_submission < ?`, oldestHour)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Delete anything past our data retention threshold, AND any timed-out KeyClaims.
func deleteOldEncryptionKeys(db *sql.DB) (int64, error) {
	res, err := db.Exec(
		fmt.Sprintf(`
			DELETE FROM encryption_keys
			WHERE  (created < (NOW() - INTERVAL %d DAY))
			OR    ((created < (NOW() - INTERVAL %d MINUTE)) AND app_public_key IS NULL)
			OR    remaining_keys = 0
		`, encryptionKeyValidityDays, oneTimeCodeExpiryInMinutes),
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func claimKey(db *sql.DB, oneTimeCode string, appPublicKey []byte) ([]byte, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	var exists int
	if err := tx.QueryRow("SELECT COUNT(*) FROM encryption_keys WHERE app_public_key = ?", appPublicKey).Scan(&exists); err != nil {
		if err := tx.Rollback(); err != nil {
			return nil, err
		}
		return nil, err
	}
	if exists == 1 {
		if err := tx.Rollback(); err != nil {
			return nil, err
		}
		return nil, ErrDuplicateKey
	}

	var created time.Time
	if err := tx.QueryRow("SELECT created FROM encryption_keys WHERE one_time_code = ?", oneTimeCode).Scan(&created); err != nil {
		if err := tx.Rollback(); err != nil {
			return nil, err
		}
		return nil, ErrInvalidOneTimeCode
	}
	created = timemath.MostRecentUTCMidnight(created)

	if created.Unix() == int64(0) {
		if err := tx.Rollback(); err != nil {
			return nil, err
		}
		return nil, ErrInvalidOneTimeCode
	}

	s, err := tx.Prepare(
		fmt.Sprintf(
			`UPDATE encryption_keys
			SET one_time_code = NULL,
				app_public_key = ?,
				created = ?
			WHERE one_time_code = ?
			AND created > (NOW() - INTERVAL %d MINUTE)`,
			oneTimeCodeExpiryInMinutes,
		),
	)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return nil, err
		}
		return nil, err
	}

	res, err := s.Exec(appPublicKey, created, oneTimeCode)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return nil, err
		}
		return nil, err
	}

	n, err := res.RowsAffected()
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return nil, err
		}
		return nil, err
	}

	if n != 1 {
		if err := tx.Rollback(); err != nil {
			return nil, err
		}
		return nil, ErrInvalidOneTimeCode
	}

	s, err = tx.Prepare(
		`SELECT server_public_key FROM encryption_keys WHERE app_public_key = ?`,
	)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return nil, err
		}
		return nil, err
	}

	row := s.QueryRow(appPublicKey)

	var serverPub []byte
	if err := row.Scan(&serverPub); err != nil {
		if err := tx.Rollback(); err != nil {
			return nil, err
		}
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return serverPub, nil
}

func persistEncryptionKey(db *sql.DB, region string, pub *[32]byte, priv *[32]byte, oneTimeCode string) error {
	_, err := db.Exec(
		`INSERT INTO encryption_keys
			(region, server_private_key, server_public_key, one_time_code, remaining_keys)
			VALUES (?, ?, ?, ?, ?)`,
		region, priv[:], pub[:], oneTimeCode, initialRemainingKeys,
	)
	return err
}

func privForPub(db *sql.DB, pub []byte) *sql.Row {
	return db.QueryRow(fmt.Sprintf(`
		SELECT server_private_key FROM encryption_keys
			WHERE server_public_key = ?
			AND created > (NOW() - INTERVAL %d DAY)
			LIMIT 1`,
		encryptionKeyValidityDays,
	),
		pub,
	)
}

// Return keys that were SUBMITTED to the Diagnosis Server during the specified
// period within the specified date.
//
// Only return keys that correspond to a Key valid for a date less than 14 days ago.
func diagnosisKeysForPeriod(db *sql.DB, period int32, currentRollingStartIntervalNumber int32) (*sql.Rows, error) {
	startHour := period
	endHour := startHour + 1
	minRollingStartIntervalNumber := timemath.RollingStartIntervalNumberPlusDays(currentRollingStartIntervalNumber, -14)

	return db.Query(
		`SELECT region, key_data, rolling_start_interval_number, rolling_period, transmission_risk_level FROM diagnosis_keys
		WHERE hour_of_submission >= ? AND hour_of_submission <= ? AND rolling_start_interval_number > ?
		ORDER BY key_data
		`, // don't implicitly order by insertion date: for privacy
		startHour, endHour, minRollingStartIntervalNumber,
	)
}

func registerDiagnosisKeys(db *sql.DB, appPubKey *[32]byte, keys []*pb.TemporaryExposureKey) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	var region string
	if err := tx.QueryRow("SELECT region FROM encryption_keys WHERE app_public_key = ?", appPubKey[:]).Scan(&region); err != nil {
		if err := tx.Rollback(); err != nil {
			return err
		}
		return err
	}

	s, err := tx.Prepare(`
		INSERT IGNORE INTO diagnosis_keys
		(region, key_data, rolling_start_interval_number, rolling_period, transmission_risk_level, hour_of_submission)
		VALUES (?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return err
		}
		return err
	}

	hourOfSubmission := timemath.HourNumber(time.Now())

	var keysInserted int64

	for _, key := range keys {
		result, err := s.Exec(region, key.GetKeyData(), key.GetRollingStartIntervalNumber(), key.GetRollingPeriod(), key.GetTransmissionRiskLevel(), hourOfSubmission)
		if err != nil {
			if err := tx.Rollback(); err != nil {
				return err
			}
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			if err := tx.Rollback(); err != nil {
				return err
			}
			return err
		}

		keysInserted += n
	}

	res, err := tx.Exec(`
		UPDATE encryption_keys
		SET remaining_keys = remaining_keys - ?
		WHERE remaining_keys >= ?
		AND app_public_key = ?`,
		keysInserted,
		keysInserted,
		appPubKey[:],
	)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return err
		}
		return ErrTooManyKeys
	}

	n, err := res.RowsAffected()
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return err
		}
		return err
	}
	if n == 0 {
		var remaining int
		if err := tx.QueryRow("SELECT remaining_keys FROM encryption_keys WHERE app_public_key = ?", appPubKey[:]).Scan(&remaining); err != nil {
			if err := tx.Rollback(); err != nil {
				return err
			}
			return err
		}
		if remaining == 0 {
			if err := tx.Rollback(); err != nil {
				return err
			}
			return ErrKeyConsumed
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}
