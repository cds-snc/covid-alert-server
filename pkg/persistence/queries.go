package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cds-snc/covid-alert-server/pkg/config"
	pb "github.com/cds-snc/covid-alert-server/pkg/proto/covidshield"
	"github.com/cds-snc/covid-alert-server/pkg/timemath"
)

func saveCountEvents(ctx context.Context, tx *sql.Tx, identifier EventType, counts []CountByOriginator) {

	for _, count := range counts {
		event := Event{
			Identifier: identifier,
			DeviceType: Server,
			Date:       time.Now(),
			Count:      count.Count,
			Originator: count.Originator,
		}
		if err := saveEvent(tx, event); err != nil {
			LogEvent(ctx, err, event)
		}
	}
}

func deleteOldDiagnosisKeys(db *sql.DB) (int64, error) {

	oldestDateNumber := timemath.DateNumber(time.Now()) - config.AppConstants.MaxDiagnosisKeyRetentionDays
	oldestHour := timemath.HourNumberAtStartOfDate(oldestDateNumber)

	res, err := db.Exec(`DELETE FROM diagnosis_keys WHERE hour_of_submission < ?`, oldestHour)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}

func deleteExpiredKeys(ctx context.Context, db *sql.DB) (int64, error) {

	tx, err := db.Begin()
	if err != nil{
		return 0, err
	}

	var (
		expiredCounts          []CountByOriginator
		expiredCountsNoUploads []CountByOriginator
		countErr               error
	)

	if expiredCounts, countErr = countExpiredClaimedEncryptionKeysByOriginator(tx); countErr != nil {
		log(ctx, countErr).Info("Unable to count expired encryption keys")
	}

	if expiredCountsNoUploads, countErr = countExpiredClaimedEncryptionKeysWithNoUploadsByOriginator(tx); countErr != nil {
		log(ctx, countErr).Info("Unable to count expired encryption keys with no uploads")
	}

	res, err := db.Exec(`
		DELETE FROM encryption_keys
		WHERE  (created < (NOW() - INTERVAL ? DAY))`,
		config.AppConstants.EncryptionKeyValidityDays,
	)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return 0, err
		}
		return 0, err
	}

	count, err := res.RowsAffected()

	if count != 0 {
		saveCountEvents(ctx, tx, OTKExpired, expiredCounts)
		saveCountEvents(ctx, tx, OTKExpiredNoUploads, expiredCountsNoUploads)
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return count, err
}

func deleteUnclaimedKeys(ctx context.Context, db *sql.DB) (int64, error) {

	//start transaction
	tx, err := db.Begin()
	if err != nil{
		return 0, err
	}

	// Count the keys we are going to delete
	var (
		unclaimedCounts        []CountByOriginator
		countErr               error
	)

	if unclaimedCounts, countErr = countUnclaimedEncryptionKeysByOriginator(tx); countErr != nil {
		log(ctx, countErr).Info("Unable to count unclaimed encryption keys")
	}

	res, err :=tx.Exec(`
		DELETE FROM encryption_keys
		WHERE created < (NOW() - INTERVAL ? MINUTE)
		AND app_public_key IS NULL`,
		config.AppConstants.OneTimeCodeExpiryInMinutes,
	)

	if err != nil {
		if err := tx.Rollback(); err != nil {
			return 0, err
		}
		return 0, err
	}

	count, err := res.RowsAffected()

	if count != 0 {
		saveCountEvents(ctx, tx, OTKUnclaimed, unclaimedCounts)
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return count, err
}

func deleteExhaustedKeys(ctx context.Context, db *sql.DB) (int64, error) {
	//start transaction
	tx, err := db.Begin()
	if err != nil{
		return 0, err
	}

	// Count the keys we are going to delete
	var (
		exhaustedCounts        []CountByOriginator
		countErr               error
	)

	if exhaustedCounts, countErr = countExhaustedEncryptionKeysByOriginator(tx); countErr != nil {
		log(ctx, countErr).Info("Unable to count exhausted encryption keys")
	}

	res, err := db.Exec(`
		DELETE FROM encryption_keys
		WHERE    remaining_keys = 0`,
	)

	if err != nil {
		if err := tx.Rollback(); err != nil {
			return 0, err
		}
		return 0, err
	}

	count, err := res.RowsAffected()

	if count != 0 {
		saveCountEvents(ctx, tx, OTKExhausted, exhaustedCounts)
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return 0, err
}

func claimKey(db *sql.DB, oneTimeCode string, appPublicKey []byte, ctx context.Context) ([]byte, error) {
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

	// we need to capture originator so that we can log it later when capturing this event
	var originator string

	row := tx.QueryRow("SELECT created, originator FROM encryption_keys WHERE one_time_code = ?", oneTimeCode)
	if err := row.Scan(&created, &originator); err != nil {

		fmt.Println(err)
		if err := tx.Rollback(); err != nil {
			return nil, err
		}
		return nil, ErrInvalidOneTimeCode
	}
	otkCreated := created
	created = timemath.MostRecentUTCMidnight(created)

	if created.Unix() == int64(0) {
		if err := tx.Rollback(); err != nil {
			return nil, err
		}
		return nil, ErrInvalidOneTimeCode
	}

	s, err := tx.Prepare(`
		UPDATE encryption_keys
		SET one_time_code = NULL,
			app_public_key = ?,
			created = ?
		WHERE one_time_code = ?
		AND created > (NOW() - INTERVAL ? MINUTE)
	`)

	if err != nil {
		if err := tx.Rollback(); err != nil {
			return nil, err
		}
		return nil, err
	}

	res, err := s.Exec(appPublicKey, created, oneTimeCode, config.AppConstants.OneTimeCodeExpiryInMinutes)
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

	row = s.QueryRow(appPublicKey)

	var serverPub []byte
	if err := row.Scan(&serverPub); err != nil {
		if err := tx.Rollback(); err != nil {
			return nil, err
		}
		return nil, err
	}

	otkDuration := OtkDuration{ originator,time.Now().Sub(otkCreated) }
	if err := saveOtkDuration(tx, otkDuration); err != nil {
		log(ctx, nil).Infof("Unable to save otkCreated %f", otkDuration.Duration.Minutes())
	}

	event := Event{Originator: originator, DeviceType: Server, Identifier: OTKClaimed, Count: 1, Date: time.Now()}
	if err := saveEvent(tx, event); err != nil {
		LogEvent(ctx, err, event)
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return serverPub, nil
}

func persistEncryptionKey(db *sql.DB, region, originator string, pub *[32]byte, priv *[32]byte, oneTimeCode string) error {
	_, err := db.Exec(
		`INSERT INTO encryption_keys
			(region, originator, server_private_key, server_public_key, one_time_code, remaining_keys)
			VALUES (?, ?, ?, ?, ?, ?)`,
		region, originator, priv[:], pub[:], oneTimeCode, config.AppConstants.InitialRemainingKeys,
	)
	return err
}

func persistEncryptionKeyWithHashID(db *sql.DB, region, originator, hashID string, pub *[32]byte, priv *[32]byte, oneTimeCode string) error {
	_, err := db.Exec(
		`INSERT INTO encryption_keys
			(region, originator, hash_id, server_private_key, server_public_key, one_time_code, remaining_keys)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
		region, originator, hashID, priv[:], pub[:], oneTimeCode, config.AppConstants.InitialRemainingKeys,
	)
	if err == nil {
		return err
	} else if strings.Contains(err.Error(), "for key 'one_time_code") { // OTC duplicate, re-run
		return err
	} else if strings.Contains(err.Error(), "for key 'hash_id") { // HashID duplicate
		var oneTimeCode sql.NullString
		row := db.QueryRow("SELECT one_time_code FROM encryption_keys WHERE hash_id = ?", hashID)
		row.Scan(&oneTimeCode)

		if oneTimeCode.Valid { // unused hashID found
			_, err = db.Exec(`DELETE FROM encryption_keys WHERE hash_id = ? AND one_time_code IS NOT NULL`, hashID)
			if err != nil {
				return err
			}
			return errors.New("regenerate OTC for hashID")
		}
		return errors.New("used hashID found")
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
func diagnosisKeysForHours(db *sql.DB, region string, startHour uint32, endHour uint32, currentRollingStartIntervalNumber int32) (*sql.Rows, error) {
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

func registerDiagnosisKeys(db *sql.DB, appPubKey *[32]byte, keys []*pb.TemporaryExposureKey, ctx context.Context) error {
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
		INSERT INTO tek_upload_count
		(originator, date, count, first_upload)
		VALUES (?, ?, ?, ?)`,
		translateToken(originator),
		time.Now().Format("2006-01-02"),
		keysInserted,
		remainingKeys == int64(config.AppConstants.InitialRemainingKeys),
	)

	if err != nil {
		if err := tx.Rollback(); err != nil {
			return err
		}
		return err
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
