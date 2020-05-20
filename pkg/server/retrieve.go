package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/CovidShield/server/pkg/persistence"
	pb "github.com/CovidShield/server/pkg/proto/covidshield"
	"github.com/CovidShield/server/pkg/retrieval"

	"github.com/sirupsen/logrus"

	"github.com/Shopify/goose/srvutil"
	"github.com/gorilla/mux"
)

const (
	numberOfDaysToServe = 14
	hoursInDay          = 24
)

type Platform int

const (
	IOS Platform = iota
	Android
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
	r.HandleFunc("/retrieve/ios/{hour:[0-9]{6}}/{auth:.*}", s.retrieveIOS)
	r.HandleFunc("/retrieve/android/{date:[0-9]{4}-[0-9]{2}-[0-9]{2}}/{auth:.*}", s.retrieveAndroid)
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

func (s *retrieveServlet) retrieveIOS(w http.ResponseWriter, r *http.Request) {
	_ = s.retrieve(w, r, IOS)
}

func (s *retrieveServlet) retrieveAndroid(w http.ResponseWriter, r *http.Request) {
	_ = s.retrieve(w, r, Android)
}

func (s *retrieveServlet) retrieve(w http.ResponseWriter, r *http.Request, platform Platform) result {
	ctx := r.Context()
	vars := mux.Vars(r)

	if !s.auth.Authenticate(vars["hour"], vars["auth"]) {
		return s.fail(log(ctx, nil), w, "invalid auth parameter", "unauthorized", http.StatusUnauthorized)
	}

	if r.Method != "GET" {
		return s.fail(log(ctx, nil).WithField("method", r.Method), w, "method not allowed", "", http.StatusMethodNotAllowed)
	}

	hourNumber64, err := strconv.ParseInt(vars["hour"], 10, 32)
	if err != nil {
		return s.fail(log(ctx, err), w, "invalid hour parameter", "", http.StatusBadRequest)
	}
	hourNumber := int32(hourNumber64)

	var keysByRegion map[string][]*pb.TemporaryExposureKey

	startTimestamp := time.Unix(int64(hourNumber*3600), 0)
	endTimestamp := time.Unix(int64((hourNumber+2)*3600), 0)

	currentRSIN := pb.CurrentRollingStartIntervalNumber()

	// TODO: reject requests for future slices
	// TODO: reject requests for current slice
	// TODO: reject requests for slices more than 14 days ago
	// TODO: verify that period is and even number

	// now := time.Now()

	// requestedDate := timemath.DateNumber(date)
	// currentDate := timemath.DateNumber(now)
	// if requestedDate != currentDate && requestedDate != currentDate-1 {
	// 	s.fail(log(ctx, err), w, "request for hour in too-old date", "use /retrieve-day for data not from today or yesterday", http.StatusNotFound)
	// 	return
	// }

	// requested := timemath.DateNumber(date)
	// current := timemath.DateNumber(time.Now())
	// if requested == current {
	// 	return s.fail(log(ctx, err), w, "request for current date", "use /retrieve-hour for today's data", http.StatusNotFound)
	// } else if requested > current {
	// 	return s.fail(log(ctx, err), w, "request for future data", "cannot request future data", http.StatusNotFound)
	// } else if requested < (current - numberOfDaysToServe) {
	// 	return s.fail(log(ctx, err), w, "request for too-old data", "requested data no longer valid", http.StatusGone)
	// }

	// if hour == currentHour {
	// 	return s.fail(log(ctx, err), w, "request for current hour", "cannot serve data for current hour for privacy reasons", http.StatusNotFound)
	// } else if hour > currentHour {
	// 	return s.fail(log(ctx, err), w, "request for future data", "cannot request future data", http.StatusNotFound)
	// }

	keysByRegion, err = s.db.FetchKeysForPeriod(hourNumber, currentRSIN)
	if err != nil {
		return s.fail(log(ctx, err), w, "database error", "", http.StatusInternalServerError)
	}

	w.Header().Add("Content-Type", "application/x-protobuf; delimited=true")
	w.Header().Add("Cache-Control", "public, max-age=3600, max-stale=600")

	// TODO new format
	if err := retrieval.SerializeTo(ctx, w, keysByRegion, startTimestamp, endTimestamp, s.signer); err != nil {
		log(ctx, err).Info("error writing response")
	}
	return result(struct{}{})
}
