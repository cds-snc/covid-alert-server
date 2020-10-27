package persistence

import (
	"context"
	"database/sql"
	"os"
)

// ClearDiagnosisKeys Truncates the diagnosis_keys table to support testing
func (c *conn) ClearDiagnosisKeys(ctx context.Context) error {
	return clearDiagnosisKeys(ctx, c.db)
}

func clearDiagnosisKeys(ctx context.Context, db *sql.DB) error {
	// This should never happen but I want to be extra sure this isn't in production
	if os.Getenv("ENABLE_TEST_TOOLS") != "true" {
		panic("un-allowed attempt to call clearDiagnosis keys")
	}

	if _, err := db.Exec(`TRUNCATE TABLE diagnosis_keys`); err != nil {
		return err
	}
	log(ctx,nil).Info("diagnosis_keys was truncated")
	return nil
}