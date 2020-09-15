package main

import (
	"github.com/Shopify/goose/logger"
	"github.com/Shopify/goose/safely"

	"github.com/cds-snc/covid-alert-server/pkg/app"
	"github.com/cds-snc/covid-alert-server/pkg/telemetry"
)

var log = logger.New("main")

func main() {
	defer safely.Recover() // panics -> bugsnag

	log(nil, nil).Info("starting")

	mainApp, db := app.NewBuilder().WithSubmission().WithRetrieval().Build()

	defer telemetry.Initialize(db).Cleanup()

	err := mainApp.RunAndWait()
	defer log(nil, err).Info("final message before shutdown")
}
