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

func NewRetrieveServlet(db persistence.Conn, auth retrieval.Authenticator) srvutil.Servlet {
	return &retrieveServlet{db: db, auth: auth}
}

type retrieveServlet struct {
	db   persistence.Conn
	auth retrieval.Authenticator
}

func (s *retrieveServlet) RegisterRouting(r *mux.Router) {
	r.HandleFunc("/retrieve-day/{date:[0-9]{4}-[0-9]{2}-[0-9]{2}}/{auth:.*}", s.retrieve)
	r.HandleFunc("/retrieve-hour/{date:[0-9]{4}-[0-9]{2}-[0-9]{2}}/{hour:[0-9]{2}}/{auth:.*}", s.retrieve)
}

func (s *retrieveServlet) fail(logger *logrus.Entry, w http.ResponseWriter, logMsg string, responseMsg string, responseCode int) {
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
}

// For a provided `?date=`, return all of the Diagnosis Keys which were
// ACCEPTED by the server on *that date* (NOT the keys that correspond to
// Keys for that date).
//
// The set of keys returned, however, only includes Keys for
// dates in the range of 0-14 days ago from NOW (though, because of the way
// client frameworks are implemented, there will never be keys for the actual
// current date).
func (s *retrieveServlet) retrieve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	if !s.auth.Authenticate(vars["date"], vars["hour"], vars["auth"]) {
		log(ctx, nil).Info("invalid auth parameter")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	date, err := time.Parse("2006-01-02", vars["date"])
	if err != nil {
		s.fail(log(ctx, err), w, "invalid date parameter", "", http.StatusBadRequest)
		return
	}

	if r.Method != "GET" {
		s.fail(log(ctx, err).WithField("method", r.Method), w, "method not allowed", "", http.StatusMethodNotAllowed)
		return
	}

	var keysByRegion map[string][]*pb.Key

	var startTimestamp time.Time
	var endTimestamp time.Time

	currentRSN := pb.CurrentRollingStartNumber()

	if _, ok := vars["hour"]; ok {
		hour, err := strconv.ParseUint(vars["hour"], 10, 32)
		if err != nil {
			s.fail(log(ctx, err), w, "invalid hour parameter", "", http.StatusBadRequest)
			return
		}

		if hour >= hoursInDay { // uint64, so can't be < 0
			s.fail(log(ctx, err), w, "invalid hour number", "", http.StatusBadRequest)
			return
		}

		{
			now := time.Now()
			requestedDate := timemath.DateNumber(date)
			currentDate := timemath.DateNumber(now)
			if requestedDate != currentDate && requestedDate != currentDate-1 {
				s.fail(log(ctx, err), w, "request for hour in too-old date", "use /retrieve-day for data not from today or yesterday", http.StatusNotFound)
				return
			}

			currentHour := uint64(timemath.HourNumber(now) - hoursInDay*requestedDate)
			if hour == currentHour {
				s.fail(log(ctx, err), w, "request for current hour", "cannot serve data for current hour for privacy reasons", http.StatusNotFound)
				return
			} else if hour > currentHour {
				s.fail(log(ctx, err), w, "request for future data", "cannot request future data", http.StatusNotFound)
				return
			}

			startTimestamp = time.Unix(int64(requestedDate*86400)+int64(hour*3600), 0)
			endTimestamp = time.Unix(int64(requestedDate*86400)+int64((hour+1)*3600), 0)
		}

		keysByRegion, err = s.db.FetchKeysForHour(date, int(hour), currentRSN)
		if err != nil {
			s.fail(log(ctx, err), w, "database error", "", http.StatusInternalServerError)
			return
		}
	} else {
		{
			requested := timemath.DateNumber(date)
			current := timemath.DateNumber(time.Now())
			if requested == current {
				s.fail(log(ctx, err), w, "request for current date", "use /retrieve-hour for today's data", http.StatusNotFound)
				return
			} else if requested > current {
				s.fail(log(ctx, err), w, "request for future data", "cannot request future data", http.StatusNotFound)
				return
			} else if requested < (current - numberOfDaysToServe) {
				s.fail(log(ctx, err), w, "request for too-old data", "requested data no longer valid", http.StatusGone)
				return
			}

			startTimestamp = time.Unix(int64(requested*86400), 0)
			endTimestamp = time.Unix(int64((requested+1)*86400), 0)
		}

		keysByRegion, err = s.db.FetchKeysForDay(date, currentRSN)
		if err != nil {
			s.fail(log(ctx, err), w, "database error", "", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Add("Content-Type", "application/x-protobuf; delimited=true")
	w.Header().Add("Cache-Control", "public, max-age=3600, max-stale=600")

	if err := retrieval.SerializeTo(ctx, w, keysByRegion, startTimestamp, endTimestamp); err != nil {
		log(ctx, err).Info("error writing response")
	}
}
