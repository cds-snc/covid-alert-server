package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/CovidShield/server/pkg/persistence"
	pb "github.com/CovidShield/server/pkg/proto/covidshield"
	"github.com/CovidShield/server/pkg/retrieval"
	"github.com/CovidShield/server/pkg/timemath"

	"github.com/sirupsen/logrus"

	"github.com/Shopify/goose/srvutil"
	"github.com/gorilla/mux"
)

const (
	numberOfDaysToServe = 14
	hoursInDay          = 24
	hoursPerPeriod      = 2
	availableHours      = numberOfDaysToServe * hoursInDay // 336
)

func NewRetrieveServlet(db persistence.Conn, auth retrieval.Authenticator, signer retrieval.Signer) srvutil.Servlet {
	return &retrieveServlet{db: db, auth: auth, signer: signer}
}

type retrieveServlet struct {
	db     persistence.Conn
	auth   retrieval.Authenticator
	signer retrieval.Signer
}

func (s *retrieveServlet) RegisterRouting(r *mux.Router) {
	// becomes 7 digits in 2084
	r.HandleFunc("/retrieve/{period:[0-9]{6}}/{auth:.*}", s.retrieveWrapper)
}

// returning this from s.fail and the s.retrieve makes it harder to call s.fail but forget to return.
type result struct{}

func (s *retrieveServlet) fail(logger *logrus.Entry, w http.ResponseWriter, logMsg string, responseMsg string, responseCode int) result {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	if responseCode == http.StatusInternalServerError {
		logger.Error(logMsg)
	} else {
		logger.Warn(logMsg)
	}
	if responseMsg == "" {
		responseMsg = logMsg
	}
	http.Error(w, responseMsg, responseCode)
	return result(struct{}{})
}

func (s *retrieveServlet) retrieveWrapper(w http.ResponseWriter, r *http.Request) {
	_ = s.retrieve(w, r)
}

func (s *retrieveServlet) retrieve(w http.ResponseWriter, r *http.Request) result {
	ctx := r.Context()
	vars := mux.Vars(r)

	if !s.auth.Authenticate(vars["period"], vars["auth"]) {
		return s.fail(log(ctx, nil), w, "invalid auth parameter", "unauthorized", http.StatusUnauthorized)
	}

	if r.Method != "GET" {
		return s.fail(log(ctx, nil).WithField("method", r.Method), w, "method not allowed", "", http.StatusMethodNotAllowed)
	}

	hourNumber64, err := strconv.ParseInt(vars["period"], 10, 32)
	if err != nil {
		return s.fail(log(ctx, err), w, "invalid period parameter", "", http.StatusBadRequest)
	}
	period := int32(hourNumber64)

	var keysByRegion map[string][]*pb.TemporaryExposureKey

	startTimestamp := time.Unix(int64(period*3600), 0)
	endTimestamp := time.Unix(int64((period+2)*3600), 0)

	currentRSIN := pb.CurrentRollingStartIntervalNumber()
	currentPeriod := timemath.CurrentPeriod()

	if period%2 != 0 {
		return s.fail(log(ctx, err), w, "odd period", "period must be even", http.StatusNotFound)
	} else if period == currentPeriod {
		return s.fail(log(ctx, err), w, "request for current period", "cannot serve data for current period for privacy reasons", http.StatusNotFound)
	} else if period > currentPeriod {
		return s.fail(log(ctx, err), w, "request for future data", "cannot request future data", http.StatusNotFound)
	} else if period < (currentPeriod - availableHours) {
		return s.fail(log(ctx, err), w, "request for too-old data", "requested data no longer valid", http.StatusGone)
	}

	// TODO: Maybe implement multi-pack linked-list scheme depending on what we hear back from G/A

	keysByRegion, err = s.db.FetchKeysForPeriod(period, currentRSIN)
	if err != nil {
		return s.fail(log(ctx, err), w, "database error", "", http.StatusInternalServerError)
	}

	w.Header().Add("Content-Type", "application/zip")
	w.Header().Add("Cache-Control", "public, max-age=3600, max-stale=600")

	// TODO new format
	if err := retrieval.SerializeTo(ctx, w, keysByRegion, startTimestamp, endTimestamp, s.signer); err != nil {
		log(ctx, err).Info("error writing response")
	}
	return result(struct{}{})
}
