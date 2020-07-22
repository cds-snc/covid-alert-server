package expiration

import (
	"context"
	"time"

	"github.com/CovidShield/server/pkg/config"
	"github.com/CovidShield/server/pkg/persistence"

	"github.com/Shopify/goose/genmain"
	"github.com/Shopify/goose/logger"
	"gopkg.in/tomb.v2"
)

var log = logger.New("expiration")

type Worker interface {
	genmain.Component
}

type worker struct {
	db       persistence.Conn
	interval time.Duration
	tomb     *tomb.Tomb
}

func (w *worker) Run() error {
	for {
		select {
		case <-w.tomb.Dying():
			return nil
		case <-time.After(w.interval):
			ctx, _ := logger.WithUUID(context.Background())
			if err := w.run(ctx); err != nil {
				log(ctx, err).Error("expiration worker failed to run")
			}
		}
	}
}

func (w *worker) run(ctx context.Context) error {
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

	if nDeleted, err := w.db.DeleteOldNonces(); err != nil {
		log(ctx, err).Info("failed to delete old nonces")
		lastErr = err
	} else {
		log(ctx, nil).WithField("count", nDeleted).Info("deleted old nonces")
	}

	return lastErr
}

func (w *worker) Tomb() *tomb.Tomb {
	return w.tomb
}

func StartWorker(db persistence.Conn) (Worker, error) {
	return create(db, time.Duration(config.AppConstants.WorkerExpirationInterval)*time.Second)
}

func create(db persistence.Conn, interval time.Duration) (Worker, error) {
	worker := &worker{
		db:       db,
		interval: interval,
		tomb:     &tomb.Tomb{},
	}

	// Run the worker once, before returning, to clean out old data on boot.
	// run will be called again in a loop by genmain
	ctx, _ := logger.WithUUID(context.Background())
	if err := worker.run(ctx); err != nil {
		return nil, err
	}

	return worker, nil
}
