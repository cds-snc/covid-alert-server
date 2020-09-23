package server

import (
	"github.com/Shopify/goose/srvutil"
	"github.com/cds-snc/covid-alert-server/pkg/config"
	"github.com/cds-snc/covid-alert-server/pkg/persistence"
	"github.com/cds-snc/covid-alert-server/pkg/retrieval"
	"github.com/gorilla/mux"
	"net/http"
)

func NewMetricsServlet(db persistence.Conn, auth retrieval.Authenticator) srvutil.Servlet {
	return &metricsServlet{db: db, auth: auth}
}

type metricsServlet struct {
	db persistence.Conn
	auth retrieval.Authenticator
}

func (m metricsServlet) RegisterRouting(r *mux.Router) {
	r.HandleFunc("/metrics/", m.metrics)
}



func (m metricsServlet) metrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {

	case http.MethodOptions:
		handleOptions(w, ctx)
		return
	case "GET":
		m.handleGet(w, r, ctx)
		return
	default:
		log(ctx, nil).WithField("method", r.Method).Info("disallowed method")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
}

func (m *metricsServlet) handleGet (w http.ResponseWriter, r *http.Request, ctx context.Context) {

	hdr := r.Header.Get("Authorization")

	region, originator, ok := m.auth.RegionFromAuthHeader(hdr)
	if !ok {
		log(ctx, nil).WithField("header", hdr).Info("bad auth header")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
}


func handleOptions(w http.ResponseWriter, ctx context.Context) {
	w.Header().Add("Access-Control-Allow-Origin", config.AppConstants.CORSAccessControlAllowOrigin)
	w.Header().Add("Access-Control-Allow-Methods", "GET")
	w.Header().Add("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Referer, User-Agent")
	if _, err := w.Write([]byte("")); err != nil {
		log(ctx, err).Warn("error writing response")
	}
	return
}
