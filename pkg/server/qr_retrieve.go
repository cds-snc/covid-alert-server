package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/cds-snc/covid-alert-server/pkg/config"
	"github.com/cds-snc/covid-alert-server/pkg/persistence"
	"github.com/cds-snc/covid-alert-server/pkg/retrieval"
	"github.com/cds-snc/covid-alert-server/pkg/timemath"

	"github.com/sirupsen/logrus"

	"github.com/Shopify/goose/srvutil"
	"github.com/gorilla/mux"
)

func NewQrRetrieveServlet(db persistence.Conn, auth retrieval.Authenticator, signer retrieval.Signer) srvutil.Servlet {
	log(nil, nil).Info("registering QR retrieval servlet")
	return &qrRetrieveServlet{db: db, auth: auth, signer: signer}
}

type qrRetrieveServlet struct {
	db     persistence.Conn
	auth   retrieval.Authenticator
	signer retrieval.Signer
}

func (s *qrRetrieveServlet) RegisterRouting(r *mux.Router) {
	// becomes 7 digits in 2084
	log(nil, nil).Info("registering QR retrieval route")
	r.HandleFunc("/qr/{region:[0-9]{3}}/{day:[0-9]{5}}/{auth:.*}", s.qrRetrieveWrapper)
}

func (s *qrRetrieveServlet) fail(logger *logrus.Entry, w http.ResponseWriter, logMsg string, responseMsg string, responseCode int) result {
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

func (s *qrRetrieveServlet) qrRetrieveWrapper(w http.ResponseWriter, r *http.Request) {
	_ = s.qrRetrieve(w, r)
}

func (s *qrRetrieveServlet) qrRetrieve(w http.ResponseWriter, r *http.Request) result {
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

	if config.AppConstants.EnableEntirePeriodBundle == true && vars["day"] == "00000" {

		endDate := timemath.CurrentDateNumber() - 1
		startDate := endDate - numberOfDaysToServe

		dateNumber = endDate

		startTimestamp = time.Unix(int64(startDate*86400), 0)
		endTimestamp = time.Unix(int64((endDate+1)*86400), 0)

	} else {

		dateNumber64, err := strconv.ParseUint(vars["day"], 10, 32)
		if err != nil {
			return s.fail(log(ctx, err), w, "invalid day parameter", "", http.StatusBadRequest)
		}
		dateNumber = uint32(dateNumber64)

		startTimestamp = time.Unix(int64(dateNumber*86400), 0)
		endTimestamp = time.Unix(int64((dateNumber+1)*86400), 0)
	}

	currentDateNumber := timemath.CurrentDateNumber()

	if config.AppConstants.DisableCurrentDateCheckFeatureFlag == false && dateNumber == currentDateNumber {
		return s.fail(log(ctx, nil), w, "request for current date", "cannot serve data for current period for privacy reasons", http.StatusNotFound)
	} else if dateNumber > currentDateNumber {
		return s.fail(log(ctx, nil), w, "request for future data", "cannot request future data", http.StatusNotFound)
	} else if dateNumber < (currentDateNumber - numberOfDaysToServe) {
		return s.fail(log(ctx, nil), w, "request for too-old data", "requested data no longer valid", http.StatusGone)
	}

	locations, err := s.db.FetchOutbreakForTimeRange(startTimestamp, endTimestamp)
	if err != nil {
		return s.fail(log(ctx, err), w, "database error", "", http.StatusInternalServerError)
	}

	w.Header().Add("Content-Type", "application/zip")
	w.Header().Add("Cache-Control", "public, max-age=3600, max-stale=600")

	size, err := retrieval.SerializeOutbreakEventsTo(ctx, w, locations, startTimestamp, endTimestamp, s.signer)
	if err != nil {
		log(ctx, err).Info("error writing response")
	}
	log(ctx, nil).WithField("unzipped-size", size).WithField("locations", len(locations)).Info("Wrote outbreak event retrieval")
	return result(struct{}{})
}
