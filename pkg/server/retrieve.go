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
	r.HandleFunc("/retrieve/{region:[0-9]{3}}/{day:[0-9]{5}}/{auth:.*}", s.retrieveWrapper)
}

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

	region := vars["region"]
	if !s.auth.Authenticate(region, vars["day"], vars["auth"]) {
		return s.fail(log(ctx, nil), w, "invalid auth parameter", "unauthorized", http.StatusUnauthorized)
	}

	if r.Method != "GET" {
		return s.fail(log(ctx, nil).WithField("method", r.Method), w, "method not allowed", "", http.StatusMethodNotAllowed)
	}

	dateNumber64, err := strconv.ParseUint(vars["day"], 10, 32)
	if err != nil {
		return s.fail(log(ctx, err), w, "invalid day parameter", "", http.StatusBadRequest)
	}
	dateNumber := uint32(dateNumber64)

	startTimestamp := time.Unix(int64(dateNumber*86400), 0)
	endTimestamp := time.Unix(int64((dateNumber+1)*86400), 0)

	currentRSIN := pb.CurrentRollingStartIntervalNumber()
	currentDateNumber := timemath.CurrentDateNumber()

	if dateNumber == currentDateNumber {
		return s.fail(log(ctx, err), w, "request for current date", "cannot serve data for current period for privacy reasons", http.StatusNotFound)
	} else if dateNumber > currentDateNumber {
		return s.fail(log(ctx, err), w, "request for future data", "cannot request future data", http.StatusNotFound)
	} else if dateNumber < (currentDateNumber - numberOfDaysToServe) {
		return s.fail(log(ctx, err), w, "request for too-old data", "requested data no longer valid", http.StatusGone)
	}

	// TODO: Maybe implement multi-pack linked-list scheme depending on what we hear back from G/A

	keys, err := s.db.FetchKeysForDateNumber(region, dateNumber, currentRSIN)
	if err != nil {
		return s.fail(log(ctx, err), w, "database error", "", http.StatusInternalServerError)
	}

	w.Header().Add("Content-Type", "application/zip")
	w.Header().Add("Cache-Control", "public, max-age=3600, max-stale=600")

	size, err := retrieval.SerializeTo(ctx, w, keys, region, startTimestamp, endTimestamp, s.signer)
	if err != nil {
		log(ctx, err).Info("error writing response")
	}
	log(ctx, nil).WithField("unzipped-size", size).WithField("keys", len(keys)).Info("Wrote retrieval")
	return result(struct{}{})
}
