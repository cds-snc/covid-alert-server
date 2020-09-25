package server

import (
	"github.com/Shopify/goose/srvutil"
	"github.com/cds-snc/covid-alert-server/pkg/config"
	"github.com/cds-snc/covid-alert-server/pkg/keyclaim"
	"github.com/cds-snc/covid-alert-server/pkg/persistence"
	"github.com/gorilla/mux"
	"github.com/google/jsonapi"
	"net/http"

	"context"
)

func NewMetricsServlet(db persistence.Conn, auth keyclaim.Authenticator) srvutil.Servlet {
	log(nil, nil).Info("registering metrics servlet")
	return &metricsServlet{db: db, auth: auth}
}

type metricsServlet struct {
	db   persistence.Conn
	auth keyclaim.Authenticator
}

func (m metricsServlet) RegisterRouting(r *mux.Router) {
	log(nil, nil).Info("registering metrics route")
	r.HandleFunc("/metrics", m.metrics)
}

func (m metricsServlet) metrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {

	case http.MethodOptions:
		log(nil, nil).Info("Getting Options")
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

func (m *metricsServlet) handleGet(w http.ResponseWriter, r *http.Request, ctx context.Context) {

	hdr := r.Header.Get("Authorization")

	region, _, ok := m.auth.RegionFromAuthHeader(hdr)

	if !ok {
		log(ctx, nil).WithField("header", hdr).Info("bad auth header")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	events, err := m.db.GetServerEventsByRegion(region)
	if err != nil {
		log(ctx, err).Errorf("issue getting events from %v", region)
		http.Error(w, "error retrieving events by bearer token", http.StatusBadRequest)
		return
	}


	w.Header().Set("Content-Type", jsonapi.MediaType)
	w.WriteHeader(http.StatusOK)
	if err := jsonapi.MarshalPayload(w, events); err != nil{
		log(ctx, err).WithField("EventResults", events).Errorf("error marshaling events")
		http.Error(w, "error building json object", http.StatusInternalServerError)
		return
	}

	return
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
