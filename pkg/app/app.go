package app

import (
	"fmt"
	"os"
	"time"

	"github.com/Shopify/goose/genmain"
	"github.com/Shopify/goose/logger"
	"github.com/Shopify/goose/srvutil"

	"github.com/CovidShield/server/pkg/expiration"
	"github.com/CovidShield/server/pkg/keyclaim"
	"github.com/CovidShield/server/pkg/persistence"
	"github.com/CovidShield/server/pkg/retrieval"
	"github.com/CovidShield/server/pkg/server"
)

var log = logger.New("app")

const (
	defaultSubmissionServerPort = 8000
	defaultRetrievalServerPort  = 8001

	defaultServerPort = 8010
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
	builder := &AppBuilder{
		defaultServerPort: defaultServerPort,
		database:          newDatabase(DatabaseURL()),
	}
	builder.servlets = append(builder.servlets, server.NewServicesServlet())
	return builder
}

func (a *AppBuilder) WithSubmission() *AppBuilder {
	a.defaultServerPort = defaultSubmissionServerPort

	a.servlets = append(a.servlets, server.NewUploadServlet(a.database))
	a.servlets = append(a.servlets, server.NewKeyClaimServlet(a.database, keyclaim.NewAuthenticator()))
	return a
}

func (a *AppBuilder) WithRetrieval() *AppBuilder {
	migrateDB(DatabaseURL()) // This is a bit of a weird place for this but it works for now.

	a.defaultServerPort = defaultRetrievalServerPort

	a.components = append(a.components, newExpirationWorker(a.database))

	a.servlets = append(a.servlets, server.NewConfigServlet())
	a.servlets = append(a.servlets, server.NewRetrieveServlet(a.database, retrieval.NewAuthenticator(), retrieval.NewSigner()))

	return a
}

func (a *AppBuilder) Build() *App {
	a.components = append(a.components, server.New(bindAddr(a.defaultServerPort), a.servlets))

	main := genmain.New(a.components...)
	main.SetShutdownDeadline(time.Duration(1) * time.Second)
	return &App{&main}
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

func newExpirationWorker(db persistence.Conn) expiration.Worker {
	worker, err := expiration.StartWorker(db)
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
