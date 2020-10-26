package app

import (
	"github.com/cds-snc/covid-alert-server/pkg/config"
	"github.com/cds-snc/covid-alert-server/pkg/server"
	"os"
)

func (a *AppBuilder) WithTestTools() *AppBuilder {


	if os.Getenv("ENABLE_TEST_TOOLS") != "true" {
		return a
	}

	log(nil, nil).Info("registering TestTools")
	a.defaultServerPort = config.AppConstants.DefaultTestToolsServerPort

	a.servlets = append(a.servlets, server.NewAdminToolsServlet(a.database, lookup))

	return a
}