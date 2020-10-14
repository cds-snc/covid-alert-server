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
	r.HandleFunc(fmt.Sprintf("/events/{startDate:%s}", DATEFORMAT), m.handleEventRequest)


}

func authorizeRequest(r *http.Request) error {

	uname, pword, ok := r.BasicAuth()
	if !ok {
		return fmt.Errorf("basic auth required for access")
	}

	metricUsername := os.Getenv("METRICS_USERNAME")
	metricPassword := os.Getenv("METRICS_PASSWORD")

	if uname != metricUsername || pword != metricPassword {
		return fmt.Errorf("invalid username or password")
	}

	return nil
}


func (m *metricsServlet) handleEventRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := authorizeRequest(r); err != nil {
		log(ctx, err).Info("Unauthorized BasicAuth")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != "GET" {
		log(ctx, nil).WithField("method", r.Method).Info("disallowed method")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	m.getEvents(ctx, w, r)
	return
}

func (m *metricsServlet) getEvents(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	startDateVal := vars["startDate"]
	_, err := time.Parse(ISODATE, startDateVal)
	if err != nil {
		log(ctx, err).Errorf("issue parsing %s", startDateVal)
		http.Error(w, "error parsing date", http.StatusBadRequest)
		return
	}

	events, err := m.db.GetServerEvents(startDateVal)
	if err != nil {
		log(ctx, err).Errorf("issue getting events")
		http.Error(w, "error retrieving events", http.StatusBadRequest)
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

	_, err = w.Write(js)
	if err != nil {
		log(ctx, err).Errorf("error writing json")
		http.Error(w, "error retrieving results", http.StatusInternalServerError)
	}
	return
}
