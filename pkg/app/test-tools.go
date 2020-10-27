package app

import (
	"github.com/cds-snc/covid-alert-server/pkg/server"
	"os"
)

func (a *AppBuilder) WithTestTools() *AppBuilder {


	if os.Getenv("ENABLE_TEST_TOOLS") != "true" {
		return a
	}

	log(nil, nil).Info("registering TestTools")

	a.servlets = append(a.servlets, server.NewTestToolsServlet(a.database, lookup))

	return a
}