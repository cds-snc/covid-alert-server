package app

import (
	"github.com/Shopify/goose/srvutil"
	"github.com/cds-snc/covid-alert-server/pkg/config"
	"github.com/cds-snc/covid-alert-server/pkg/server"
	"github.com/cds-snc/covid-alert-server/pkg/testhelpers"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestAppBuilder_WithTestTools_EnableTestToolsIsDisabled(t *testing.T) {

	_, oldLog := testhelpers.SetupTestLogging(&log)
	defer func() { log = *oldLog }()

	app := AppBuilder{
		defaultServerPort: 0,
		components:        nil,
		servlets:          []srvutil.Servlet{},
		database:          nil,
	}

	app.WithTestTools()

	expected := append([]srvutil.Servlet{})
	assert.ElementsMatch(t, expected, app.servlets, "should not register adminToolsServlet if ENABLE_TEST_TOOLS is not true")
}

func TestAppBuilder_WithTestTools_EnableTestToolsIsEnabled(t *testing.T) {

	_, oldLog := testhelpers.SetupTestLogging(&log)
	defer func () { log = *oldLog }()

	app := AppBuilder{
		defaultServerPort: 0,
		components:        nil,
		servlets:          []srvutil.Servlet{},
		database:          nil,
	}

	os.Setenv("ENABLE_TEST_TOOLS","true")
	app.WithTestTools()

	expected := append([]srvutil.Servlet{server.NewAdminToolsServlet(nil, nil)})
	assert.ElementsMatch(t, expected, app.servlets, "should register the adminToolsServlet if ENABLE_TEST_TOOLS is true")
	assert.Equal(t, app.defaultServerPort, config.AppConstants.DefaultTestToolsServerPort, "should set default port")
}

