package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Shopify/goose/srvutil"
	"github.com/cds-snc/covid-alert-server/pkg/keyclaim"
	"github.com/cds-snc/covid-alert-server/pkg/persistence"
	"github.com/gorilla/mux"

	"context"
)

const ISODATE string = "2006-01-02"

func NewMetricsServlet(db persistence.Conn, auth keyclaim.Authenticator) srvutil.Servlet {
	log(nil, nil).Info("registering metrics servlet")
	return &metricsServlet{db: db, auth: auth}
}

type metricsServlet struct {
	db   persistence.Conn
	auth keyclaim.Authenticator
}

const DATEFORMAT string = "\\d{4,4}-\\d{2,2}-\\d{2,2}"
func (m metricsServlet) RegisterRouting(r *mux.Router) {
	log(nil, nil).Info("registering metrics route")
	r.HandleFunc(fmt.Sprintf("/claimed-keys/{startDate:%v}", DATEFORMAT), m.claimedKeys)

	r.HandleFunc(fmt.Sprintf("/generated-keys/{startDate:%v}", DATEFORMAT), m.generatedKeys)
	r.HandleFunc(fmt.Sprintf("/regenerated-keys/{startDate:%v}", DATEFORMAT), m.regeneratedKeys)
	r.HandleFunc(fmt.Sprintf("/expired-keys/{startDate:%v}", DATEFORMAT), m.expiredKeys)
	r.HandleFunc(fmt.Sprintf("/exhausted-keys/{startDate:%v}", DATEFORMAT), m.exhaustedKeys)
	r.HandleFunc(fmt.Sprintf("/unclaimed-keys/{startDate:%v}", DATEFORMAT), m.unclaimedKeys)

}

func authorizeRequest(r *http.Request) error {

	uname, pword, ok := r.BasicAuth()
	if !ok {
		return fmt.Errorf("basic auth required for access")
	}

	metricUsername, uok := os.LookupEnv("METRICS_USERNAME")
	if !uok {
		log(nil, nil).Panic("Metrics username not set")
	}

	metricPassword, pok := os.LookupEnv("METRICS_PASSWORD")
	if !pok {
		log(nil, nil).Panic("Metrics username not set")
	}
	if uname != metricUsername || pword != metricPassword {
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

func (m *metricsServlet) regeneratedKeys(w http.ResponseWriter, r *http.Request) {
	m.handleEventRequest(w, r, persistence.OTKRegenerated)
}

func (m *metricsServlet) expiredKeys(w http.ResponseWriter, r *http.Request) {
	m.handleEventRequest(w, r, persistence.OTKExpired)
}

func (m *metricsServlet) exhaustedKeys(w http.ResponseWriter, r *http.Request) {
	m.handleEventRequest(w, r, persistence.OTKExhausted)
}

func (m *metricsServlet) unclaimedKeys(w http.ResponseWriter, r *http.Request) {
	m.handleEventRequest(w, r, persistence.OTKUnclaimed)
}

func (m *metricsServlet) handleEventRequest(w http.ResponseWriter, r *http.Request, eventType persistence.EventType) {
	ctx := r.Context()

	if err := authorizeRequest(r); err != nil {
		log(ctx, err).Info("Unauthorized BasicAuth")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}

	if r.Method != "GET" {
		log(ctx, nil).WithField("method", r.Method).Info("disallowed method")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	m.getEvents(ctx, eventType, w, r)
	return
}

func (m *metricsServlet) getEvents(ctx context.Context, eventType persistence.EventType, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	startDateVal := vars["startDate"]
	_, err := time.Parse(ISODATE, startDateVal)
	if err != nil {
		log(ctx, err).Errorf("issue parsing %v", startDateVal)
		http.Error(w, "error parsing start date", http.StatusBadRequest)
		return
	}

	events, err := m.db.GetServerEventsByType(eventType, startDateVal)
	if err != nil {
		log(ctx, err).Errorf("issue getting events for %v", eventType)
		http.Error(w, "error retrieving events by bearer token", http.StatusBadRequest)
		return
	}

	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	log(nil,nil).Infof("Events is %v", events)
	js, err := json.Marshal(events)
	if err != nil {
		log(ctx, err).WithField("EventResults", events).Errorf("error marshaling events")
		http.Error(w, "error building json object", http.StatusInternalServerError)
		return
	}

	log(nil,nil).Infof("JS is %v", js)
	w.Write(js)
	return
}
