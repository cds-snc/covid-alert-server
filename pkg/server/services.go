package server

import (
	"encoding/json"
	"net/http"

	"github.com/Shopify/goose/srvutil"
	"github.com/gorilla/mux"
)

var branch string
var revision string

func NewServicesServlet() srvutil.Servlet {
	s := &servicesServlet{}
	return srvutil.PrefixServlet(s, "/services")
}

type servicesServlet struct{}
type version struct {
	Branch   string `json:"branch"`
	Revision string `json:"revision"`
}

func (s *servicesServlet) RegisterRouting(r *mux.Router) {
	r.HandleFunc("/ping", s.ping)
	r.HandleFunc("/present", s.exposurePresence)
	r.HandleFunc("/version.json", s.version)
}

func (s *servicesServlet) exposurePresence(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (s *servicesServlet) ping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Add("Cache-Control", "no-store")
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	if _, err := w.Write([]byte("OK\n")); err != nil {
		log(ctx, err).Info("error writing response")
	}
}

func (s *servicesServlet) version(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	w.Header().Add("Cache-Control", "no-store")
	w.Header().Add("Content-Type", "application/json; charset=utf-8")

	version := version{
		Branch:   branch,
		Revision: revision,
	}

	js, err := json.Marshal(version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(js); err != nil {
		log(ctx, err).Info("error writing response")
	}
}
