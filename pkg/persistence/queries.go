package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/CovidShield/server/pkg/config"
	pb "github.com/CovidShield/server/pkg/proto/covidshield"
	"github.com/CovidShield/server/pkg/timemath"
)

func deleteOldDiagnosisKeys(db *sql.DB) (int64, error) {
	oldestDateNumber := timemath.DateNumber(time.Now()) - config.AppConstants.MaxDiagnosisKeyRetentionDays
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
		`, config.AppConstants.EncryptionKeyValidityDays, config.AppConstants.OneTimeCodeExpiryInMinutes),
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
			config.AppConstants.OneTimeCodeExpiryInMinutes,
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

func persistEncryptionKey(db *sql.DB, region, originator, hashID string, pub *[32]byte, priv *[32]byte, oneTimeCode string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	if len(hashID) > 0 {
		var one_time_code sql.NullString
		row := tx.QueryRow("SELECT one_time_code FROM encryption_keys WHERE hash_id = ? FOR UPDATE", hashID)

		switch err := row.Scan(&one_time_code); {
		case err == sql.ErrNoRows: // no hashID found
		case err != nil:
			log(nil, err).Error(err.Error())
			if err := tx.Rollback(); err != nil {
				return err
			}
			return err
		case !one_time_code.Valid: // used hashID found
			if err := tx.Rollback(); err != nil {
				return err
			}
			return errors.New("used hashID found")
		case one_time_code.Valid: // un-used hashID found
			_, err = tx.Exec(`DELETE FROM encryption_keys WHERE hash_id = ? AND one_time_code IS NOT NULL`, hashID)
			if err != nil {
				if err := tx.Rollback(); err != nil {
					return err
				}
				return err
			}
		default:
		}
	}
	
	_, err = tx.Exec(
		`INSERT INTO encryption_keys
			(region, originator, hash_id, server_private_key, server_public_key, one_time_code, remaining_keys)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
		region, originator, hashID, priv[:], pub[:], oneTimeCode, config.AppConstants.InitialRemainingKeys,
	)
	
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return err
		}
		return err
	}
	
	if err = tx.Commit(); err != nil {
		return err
	}

	return err
}

func privForPub(db *sql.DB, pub []byte) *sql.Row {
	return db.QueryRow(fmt.Sprintf(`
		SELECT server_private_key FROM encryption_keys
			WHERE server_public_key = ?
			AND created > (NOW() - INTERVAL %d DAY)
			LIMIT 1`,
		config.AppConstants.EncryptionKeyValidityDays,
	),
		pub,
	)
}

// Return keys that were SUBMITTED to the Diagnosis Server during the specified
// UTC date.
//
// Only return keys that correspond to a Key valid for a date less than 14 days ago.
//
// TODO: this might be the right place to pad inappropriately small batches
func diagnosisKeysForDateNumber(db *sql.DB, region string, dateNumber uint32, currentRollingStartIntervalNumber int32) (*sql.Rows, error) {
	startHour := dateNumber * 24
	endHour := startHour + 24
	minRollingStartIntervalNumber := timemath.RollingStartIntervalNumberPlusDays(currentRollingStartIntervalNumber, -14)

	return db.Query(
		`SELECT region, key_data, rolling_start_interval_number, rolling_period, transmission_risk_level FROM diagnosis_keys
		WHERE hour_of_submission >= ?
		AND hour_of_submission < ?
		AND rolling_start_interval_number > ?
		AND region = ?
		ORDER BY key_data
		`, // don't implicitly order by insertion date: for privacy
		startHour, endHour, minRollingStartIntervalNumber, region,
	)
}

func registerDiagnosisKeys(db *sql.DB, appPubKey *[32]byte, keys []*pb.TemporaryExposureKey) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	var region string
	var originator string
	var remainingKeys int64
	if err := tx.QueryRow("SELECT region, originator, remaining_keys FROM encryption_keys WHERE app_public_key = ? FOR UPDATE", appPubKey[:]).Scan(&region, &originator, &remainingKeys); err != nil {
		if err := tx.Rollback(); err != nil {
			return err
		}
		return err
	}

	if remainingKeys == 0 {
		if err := tx.Rollback(); err != nil {
			return err
		}
		return ErrKeyConsumed
	}

	s, err := tx.Prepare(`
		INSERT IGNORE INTO diagnosis_keys
		(region, originator, key_data, rolling_start_interval_number, rolling_period, transmission_risk_level, hour_of_submission)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
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
		result, err := s.Exec(region, originator, key.GetKeyData(), key.GetRollingStartIntervalNumber(), key.GetRollingPeriod(), key.GetTransmissionRiskLevel(), hourOfSubmission)
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

	if remainingKeys < keysInserted {
		if err := tx.Rollback(); err != nil {
			return err
		}
		return ErrTooManyKeys
	}

	_, err = tx.Exec(`
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

	if err = tx.Commit(); err != nil {
		return err
	}
	log(nil, nil).WithField("keys", keysInserted).Info("Inserted keys")
	return nil
}

type queryRower interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}

func checkClaimKeyBan(db queryRower, identifier string) (triesRemaining int, banDuration time.Duration, err error) {
	var failures uint16
	var lastFailure time.Time
	var maxConsecutiveClaimKeyFailures = config.AppConstants.MaxConsecutiveClaimKeyFailures
	q := db.QueryRow(`SELECT failures, last_failure FROM failed_key_claim_attempts WHERE identifier = ?`, identifier)
	if err := q.Scan(&failures, &lastFailure); err != nil {
		if err.Error() == "sql: no rows in result set" {
			return maxConsecutiveClaimKeyFailures, 0, nil
		}
		return 0, 0, err
	}

	triesRemaining = maxConsecutiveClaimKeyFailures - int(failures)
	if triesRemaining < 0 {
		triesRemaining = 0
	}
	banDuration = time.Duration(0)
	if triesRemaining == 0 {
		elapsed := time.Since(lastFailure)
		banDuration = (time.Duration(config.AppConstants.ClaimKeyBanDuration) * time.Hour) - elapsed
	}

	if banDuration < time.Duration(0) {
		return maxConsecutiveClaimKeyFailures, 0, nil
	}

	return triesRemaining, banDuration, nil
}

func registerClaimKeySuccess(db *sql.DB, identifier string) error {
	_, err := db.Exec(`DELETE FROM failed_key_claim_attempts WHERE identifier = ?`, identifier)
	return err
}

func registerClaimKeyFailure(db *sql.DB, identifier string) (triesRemaining int, banDuration time.Duration, err error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, 0, err
	}

	if _, err := tx.Exec(`
		INSERT INTO failed_key_claim_attempts (identifier) VALUES (?)
		ON DUPLICATE KEY UPDATE
      failures = failures + 1,
			last_failure = NOW()
	`, identifier); err != nil {
		if err := tx.Rollback(); err != nil {
			return 0, 0, err
		}
		return 0, 0, err
	}

	triesRemaining, banDuration, err = checkClaimKeyBan(tx, identifier)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return 0, 0, err
		}
		return 0, 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	return triesRemaining, banDuration, nil
}

func deleteOldFailedClaimKeyAttempts(db *sql.DB) (int64, error) {
	threshold := time.Now().Add(-(time.Duration(config.AppConstants.ClaimKeyBanDuration) * time.Hour))

	res, err := db.Exec(`DELETE FROM failed_key_claim_attempts WHERE last_failure < ?`, threshold)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func countClaimedOneTimeCodes(db *sql.DB) (int64, error) {
	var count int64

	row := db.QueryRow("SELECT COUNT(*) FROM encryption_keys WHERE one_time_code IS NULL")
	err := row.Scan(&count)

	if err != nil {
		return -1, err
	}

	return count, err
}

func countDiagnosisKeys(db *sql.DB) (int64, error) {
	var count int64

	row := db.QueryRow("SELECT COUNT(*) FROM diagnosis_keys")
	err := row.Scan(&count)

	if err != nil {
		return -1, err
	}

	return count, err
}

func countUnclaimedOneTimeCodes(db *sql.DB) (int64, error) {
	var count int64

	row := db.QueryRow("SELECT COUNT(*) FROM encryption_keys WHERE one_time_code IS NOT NULL")
	err := row.Scan(&count)

	if err != nil {
		return -1, err
	}

	return count, err
}