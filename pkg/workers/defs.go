package workers

import (
	"context"
	"time"

	"github.com/CovidShield/server/pkg/persistence"
	"github.com/Shopify/goose/genmain"
	"github.com/Shopify/goose/logger"
	"gopkg.in/tomb.v2"
)

var log = logger.New("workers")

type Worker interface {
	genmain.Component
}

type worker struct {
	name     string
	db       persistence.Conn
	interval time.Duration
	tomb     *tomb.Tomb
	runner   func(w *worker, ctx context.Context) error
}

func (w *worker) Run() error {
	for {
		select {
		case <-w.tomb.Dying():
			return nil
		case <-time.After(w.interval):
			ctx, _ := logger.WithUUID(context.Background())
			if err := w.runner(w, ctx); err != nil {
				log(ctx, err).WithField("name", w.name).Error("worker failed to run")
			}
		}
	}
}

func (w *worker) Tomb() *tomb.Tomb {
	return w.tomb
}
