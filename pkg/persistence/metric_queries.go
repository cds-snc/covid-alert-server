package persistence

import (
	"database/sql"
	"fmt"

	"github.com/cds-snc/covid-alert-server/pkg/config"
)

func countUnclaimedEncryptionKeysByOriginator(tx *sql.Tx) ([]CountByOriginator, error) {
	return countByOriginator(tx, fmt.Sprintf(`
			SELECT originator, count(*) FROM encryption_keys
			WHERE  ((created < (NOW() - INTERVAL %d MINUTE)) AND app_public_key IS NULL)
			GROUP BY encryption_keys.originator `, config.AppConstants.OneTimeCodeExpiryInMinutes))
}

func countExpiredClaimedEncryptionKeysByOriginator(tx *sql.Tx) ([]CountByOriginator, error) {
	return countByOriginator(tx, fmt.Sprintf(`
			SELECT originator, COUNT(*) FROM encryption_keys
			WHERE  (created < (NOW() - INTERVAL %d DAY))
			GROUP BY encryption_keys.originator
		`, config.AppConstants.EncryptionKeyValidityDays))
}

func countExpiredClaimedEncryptionKeysWithNoUploadsByOriginator(tx *sql.Tx) ([]CountByOriginator, error) {
	return countByOriginator(tx, fmt.Sprintf(`
			SELECT originator, COUNT(*) FROM encryption_keys
			WHERE  (created < (NOW() - INTERVAL %d DAY)) AND remaining_keys = %d
			GROUP BY encryption_keys.originator
		`, config.AppConstants.EncryptionKeyValidityDays, config.AppConstants.InitialRemainingKeys))
}

func countExhaustedEncryptionKeysByOriginator(tx *sql.Tx) ([]CountByOriginator, error) {
	return countByOriginator(tx, `
			SELECT originator, COUNT(*) FROM encryption_keys
			WHERE  remaining_keys = 0
			GROUP BY encryption_keys.originator
		`)
}

func countByOriginator(tx *sql.Tx, query string) ([]CountByOriginator, error) {

	rows, err := tx.Query(query)
	if err != nil {
		return nil, err
	}

	var counts []CountByOriginator
	for rows.Next() {
		var (
			numberToDelete int
			originator     string
		)

		if err := rows.Scan(&originator, &numberToDelete); err != nil {
			return nil, err
		}

		counts = append(counts, CountByOriginator{
			Originator: originator,
			Count:      numberToDelete,
		})
	}

	return counts, nil
}

// CountByOriginator Just a count of a thing by the Originator (Bearer Token)
// Originator The originator (Bearer Token) of this thing we are counting
// Count The number of times this thing happened
type CountByOriginator struct {
	Originator string
	Count      int
}
