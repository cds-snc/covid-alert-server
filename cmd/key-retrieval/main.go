package main

import (
	"github.com/Shopify/goose/logger"
	"github.com/Shopify/goose/safely"

	"github.com/CovidShield/server/pkg/app"
	"github.com/CovidShield/server/pkg/telemetry"
)

var log = logger.New("main")

func main() {
	defer safely.Recover() // panics -> bugsnag

	log(nil, nil).Info("starting")

	defer telemetry.Initialize().Cleanup()

	mainApp := app.NewBuilder().WithRetrieval().Build()

	err := mainApp.RunAndWait()
	defer log(nil, err).Info("final message before shutdown")
}
