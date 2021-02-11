package app

import (
	"fmt"
	"os"
	"time"

	"github.com/Shopify/goose/genmain"
	"github.com/Shopify/goose/logger"
	"github.com/Shopify/goose/srvutil"

	"github.com/cds-snc/covid-alert-server/pkg/config"
	"github.com/cds-snc/covid-alert-server/pkg/keyclaim"
	"github.com/cds-snc/covid-alert-server/pkg/persistence"
	"github.com/cds-snc/covid-alert-server/pkg/retrieval"
	"github.com/cds-snc/covid-alert-server/pkg/server"
	"github.com/cds-snc/covid-alert-server/pkg/workers"
)

var (
	log    = logger.New("app")
	lookup keyclaim.Authenticator
)

type App struct {
	*genmain.Main
}

type AppBuilder struct {
	defaultServerPort uint32
	components        []genmain.Component
	servlets          []srvutil.Servlet
	database          persistence.Conn
}

func NewBuilder() *AppBuilder {
	config.InitConfig() // read configuration into a structure

	lookup = keyclaim.NewAuthenticator()
	persistence.SetupLookup(lookup)

	builder := &AppBuilder{
		defaultServerPort: config.AppConstants.DefaultServerPort,
		database:          newDatabase(DatabaseURL()),
	}
	builder.servlets = append(builder.servlets, server.NewServicesServlet())
	return builder
}

func checkEnvironmentVariable(key string) {

	uname, ok := os.LookupEnv(key)

	if !ok {
		panic(fmt.Sprintf("%s was not set", key))
	}

	if len(uname) < 10 {
		panic(fmt.Sprintf("%s is too short needs to be >= 10 characters", key))
	}
}

func (a *AppBuilder) WithSubmission() *AppBuilder {

	a.defaultServerPort = config.AppConstants.DefaultSubmissionServerPort

	a.servlets = append(a.servlets, server.NewUploadServlet(a.database))
	a.servlets = append(a.servlets, server.NewKeyClaimServlet(a.database, lookup))
	a.servlets = append(a.servlets, server.NewQRSubmissionServlet(a.database, lookup))

	return a
}

func (a *AppBuilder) WithRetrieval() *AppBuilder {
	migrateDB(DatabaseURL()) // This is a bit of a weird place for this but it works for now.

	a.defaultServerPort = config.AppConstants.DefaultRetrievalServerPort

	a.components = append(a.components, newExpirationWorker(a.database))

	a.servlets = append(a.servlets, server.NewRetrieveServlet(a.database, retrieval.NewAuthenticator(), retrieval.NewSigner()))

	//Check Metric existence ENV Variables
	checkEnvironmentVariable("METRICS_USERNAME")
	checkEnvironmentVariable("METRICS_PASSWORD")
	a.servlets = append(a.servlets, server.NewMetricsServlet(a.database, lookup))

	return a
}

func (a *AppBuilder) WithDefaultPort() *AppBuilder {
	a.defaultServerPort = 8000
	return a
}

func (a *AppBuilder) Build() (*App, persistence.Conn) {
	a.components = append(a.components, server.New(bindAddr(a.defaultServerPort), a.servlets))

	main := genmain.New(a.components...)
	main.SetShutdownDeadline(time.Duration(1) * time.Second)
	return &App{&main}, a.database
}

func DatabaseURL() string {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		panic("DATABASE_URL must be set")
	}
	return url
}

func newDatabase(dbURL string) persistence.Conn {
	db, err := persistence.Dial(dbURL)
	fatalIfErr(err, "could not create db object")

	return db
}

func bindAddr(defaultPort uint32) string {
	if bindAddr := os.Getenv("BIND_ADDR"); bindAddr != "" {
		return bindAddr
	}
	if port := os.Getenv("PORT"); port != "" {
		return "0.0.0.0:" + port
	}
	return fmt.Sprintf("0.0.0.0:%d", defaultPort)
}

func newExpirationWorker(db persistence.Conn) workers.Worker {
	worker, err := workers.StartExpirationWorker(db)
	fatalIfErr(err, "failed to do initial run of expiration worker")
	return worker
}

func fatalIfErr(err error, msg string) {
	if err != nil {
		log(nil, err).Fatal(msg)
	}
}

func migrateDB(databaseURL string) {
	log(nil, nil).Info("running database bootstrap / migrations")
	err := persistence.MigrateDatabase(databaseURL)
	if err != nil {
		log(nil, err).Fatal("error running database bootstrap / migrations")
	}
}
