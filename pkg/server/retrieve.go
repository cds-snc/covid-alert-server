package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/CovidShield/server/pkg/config"
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

	/* Hardcode the region as 302 (Canada MCC)
	You can see the reason for this in pkg/server/keyclaim.go
	As stated there I'm going to open an issue to continue this work instead of just
	relying on the hardcoded value.
	*/
	region := config.AppConstants.RegionCode
	if !s.auth.Authenticate(region, vars["day"], vars["auth"]) {
		return s.fail(log(ctx, nil), w, "invalid auth parameter", "unauthorized", http.StatusUnauthorized)
	}

	if r.Method != "GET" {
		return s.fail(log(ctx, nil).WithField("method", r.Method), w, "method not allowed", "", http.StatusMethodNotAllowed)
	}

	var startTimestamp time.Time
	var endTimestamp time.Time
	var dateNumber uint32
	var startHour uint32
	var endHour uint32

	if config.AppConstants.EnableEntirePeriodBundle == true && vars["day"] == "00000" {

		endDate := timemath.CurrentDateNumber() - 1
		startDate := endDate - numberOfDaysToServe

		dateNumber = endDate

		startTimestamp = time.Unix(int64(startDate*86400), 0)
		endTimestamp = time.Unix(int64((endDate+1)*86400), 0)

		startHour = startDate * 24
		endHour = (endDate * 24) + 24

	} else {

		dateNumber64, err := strconv.ParseUint(vars["day"], 10, 32)
		if err != nil {
			return s.fail(log(ctx, err), w, "invalid day parameter", "", http.StatusBadRequest)
		}
		dateNumber = uint32(dateNumber64)

		startTimestamp = time.Unix(int64(dateNumber*86400), 0)
		endTimestamp = time.Unix(int64((dateNumber+1)*86400), 0)

		startHour = dateNumber * 24
		endHour = startHour + 24

	}

	currentRSIN := pb.CurrentRollingStartIntervalNumber()
	currentDateNumber := timemath.CurrentDateNumber()

	if config.AppConstants.DisableCurrentDateCheckFeatureFlag == false && dateNumber == currentDateNumber {
		return s.fail(log(ctx, nil), w, "request for current date", "cannot serve data for current period for privacy reasons", http.StatusNotFound)
	} else if dateNumber > currentDateNumber {
		return s.fail(log(ctx, nil), w, "request for future data", "cannot request future data", http.StatusNotFound)
	} else if dateNumber < (currentDateNumber - numberOfDaysToServe) {
		return s.fail(log(ctx, nil), w, "request for too-old data", "requested data no longer valid", http.StatusGone)
	}

	keys, err := s.db.FetchKeysForHours(region, startHour, endHour, currentRSIN)
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
