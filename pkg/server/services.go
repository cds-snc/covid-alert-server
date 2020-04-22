package server

import (
	"net/http"

	"github.com/Shopify/goose/srvutil"
	"github.com/gorilla/mux"
)

func NewServicesServlet() srvutil.Servlet {
	s := &servicesServlet{}
	return srvutil.PrefixServlet(s, "/services")
}

type servicesServlet struct{}

func (s *servicesServlet) RegisterRouting(r *mux.Router) {
	r.HandleFunc("/ping", s.ping)
}

func (s *servicesServlet) ping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	if _, err := w.Write([]byte("OK\n")); err != nil {
		log(ctx, err).Info("error writing response")
	}
}
