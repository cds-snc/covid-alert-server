package workers

import (
	"context"
	"time"

	"github.com/cds-snc/covid-alert-server/pkg/config"
	"github.com/cds-snc/covid-alert-server/pkg/persistence"

	"github.com/Shopify/goose/logger"
	"gopkg.in/tomb.v2"
)

var expirationRunner = func(w *worker, ctx context.Context) error {
	log(ctx, nil).Info("running")

	var lastErr error

	if nDeleted, err := w.db.DeleteOldDiagnosisKeys(); err != nil {
		log(ctx, err).Info("failed to delete old diagnosis keys")
		lastErr = err
	} else {
		log(ctx, nil).WithField("count", nDeleted).Info("deleted old diagnosis keys")
	}

	if nDeleted, err := w.db.DeleteOldEncryptionKeys(); err != nil {
		log(ctx, err).Info("failed to delete old encryption keys")
		lastErr = err
	} else {
		log(ctx, nil).WithField("count", nDeleted).Info("deleted old encryption keys")
	}

	if nDeleted, err := w.db.DeleteOldFailedClaimKeyAttempts(); err != nil {
		log(ctx, err).Info("failed to delete old failed claim-key attempts")
		lastErr = err
	} else {
		log(ctx, nil).WithField("count", nDeleted).Info("deleted old claim-key attempts")
	}

	return lastErr
}

func StartExpirationWorker(db persistence.Conn) (Worker, error) {
	return createExpirationWorker(db, time.Duration(config.AppConstants.WorkerExpirationInterval)*time.Second)
}

func createExpirationWorker(db persistence.Conn, interval time.Duration) (Worker, error) {
	worker := &worker{
		name:     "expiration",
		db:       db,
		interval: interval,
		tomb:     &tomb.Tomb{},
		runner:   expirationRunner,
	}

	// Run the worker once, before returning, to clean out old data on boot.
	// run will be called again in a loop by genmain
	ctx, _ := logger.WithUUID(context.Background())
	if err := worker.runner(worker, ctx); err != nil {
		return nil, err
	}

	return worker, nil
}
