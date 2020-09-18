package server

import (
	"github.com/cds-snc/covid-alert-server/pkg/config"
	"github.com/gorilla/mux"
	"os"
	"testing"
)

func Router() *mux.Router {
	router := mux.NewRouter()
	return router
}

func GetPaths(router *mux.Router) []string {
	var paths []string
	router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, err := route.GetPathTemplate()
		if err != nil {
			return err
		}
		paths = append(paths, path)
		return nil
	})
	return paths
}

// TestMain this gets called instead of the regular testing main method and allows us to run setup code
func TestMain(m *testing.M)  {

	// We need to run init config before any of the server tests
	config.InitConfig()
	os.Exit(m.Run())
}