package server

import (
	"encoding/json"
	"fmt"
	"github.com/Shopify/goose/srvutil"
	"github.com/cds-snc/covid-alert-server/pkg/config"
	"github.com/cds-snc/covid-alert-server/pkg/keyclaim"
	"github.com/cds-snc/covid-alert-server/pkg/persistence"
	"github.com/gorilla/mux"
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
	r.HandleFunc("/claimed-keys", m.claimedKeys)
	r.HandleFunc("/generated-keys", m.generatedKeys)
	r.HandleFunc("/expired-keys", m.expiredKeys)
}

func authorizeRequest (r *http.Request) error {

	uname, pword, ok := r.BasicAuth()
	if !ok {
		return fmt.Errorf("basic auth required for access")
	}

	if uname != "foo" || pword != "bar" {
		return fmt.Errorf("invalid username or password")
	}

	return nil
}

func (m *metricsServlet) claimedKeys(w http.ResponseWriter, r *http.Request) {
	m.handleEventRequest(w, r, persistence.OTKClaimed)
}

func (m *metricsServlet) generatedKeys(w http.ResponseWriter, r *http.Request) {
	m.handleEventRequest(w, r, persistence.OTKGenerated)
}

func (m *metricsServlet) expiredKeys(w http.ResponseWriter, r *http.Request) {
	m.handleEventRequest(w, r, persistence.OTKExpired)
}

func (m *metricsServlet) handleEventRequest(w http.ResponseWriter, r *http.Request, eventType persistence.EventType) {
	ctx := r.Context()

	if err := authorizeRequest(r); err != nil {
		log(ctx, err).Info("Unauthorized BasicAuth")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}

	switch r.Method {

	case http.MethodOptions:
		handleOptions(ctx, w)
		return
	case "GET":
		m.getEvents(ctx, eventType, w)
		return
	default:
		log(ctx, nil).WithField("method", r.Method).Info("disallowed method")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
}

func (m *metricsServlet) getEvents(ctx context.Context, eventType persistence.EventType, w http.ResponseWriter) {
	events, err := m.db.GetServerEventsByType(eventType)
	if err != nil {
		log(ctx, err).Errorf("issue getting events for %v", eventType)
		http.Error(w, "error retrieving events by bearer token", http.StatusBadRequest)
		return
	}

	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	js, err := json.Marshal(events)
	if err != nil {
		log(ctx, err).WithField("EventResults", events).Errorf("error marshaling events")
		http.Error(w, "error building json object", http.StatusInternalServerError)
		return
	}
	w.Write(js)
	return
}

func handleOptions(ctx context.Context, w http.ResponseWriter) {
	w.Header().Add("Access-Control-Allow-Origin", config.AppConstants.CORSAccessControlAllowOrigin)
	w.Header().Add("Access-Control-Allow-Methods", "GET")
	w.Header().Add("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Referer, User-Agent")
	if _, err := w.Write([]byte("")); err != nil {
		log(ctx, err).Warn("error writing response")
	}
	return
}
