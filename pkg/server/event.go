package server

import (
	b64 "encoding/base64"
	"io/ioutil"
	"net/http"

	"github.com/CovidShield/server/pkg/config"
	"github.com/CovidShield/server/pkg/persistence"
	pb "github.com/CovidShield/server/pkg/proto/covidshield"
	"google.golang.org/protobuf/proto"

	"github.com/Shopify/goose/srvutil"
	"github.com/gorilla/mux"
)

func NewEventServlet(db persistence.Conn) srvutil.Servlet {
	return &eventServlet{db: db}
}

type eventServlet struct {
	db persistence.Conn
}

func (s *eventServlet) RegisterRouting(r *mux.Router) {
	r.HandleFunc("/event", s.event)
	r.HandleFunc("/event/nonce", s.eventNonce)
}

func eventError(errCode pb.EventResponse_ErrorCode) *pb.EventResponse {
	return &pb.EventResponse{Error: &errCode}
}

func (s *eventServlet) event(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != "POST" {
		log(ctx, nil).WithField("method", r.Method).Info("disallowed method")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Add("Content-Type", "application/x-protobuf")

	reader := http.MaxBytesReader(w, r.Body, 1024)
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		requestError(
			ctx, w, err, "error reading request",
			http.StatusBadRequest, eventError(pb.EventResponse_INVALID_DATA),
		)
		return
	}

	var event pb.EventRequest
	if err := proto.Unmarshal(data, &event); err != nil {
		requestError(
			ctx, w, err, "error unmarshalling request",
			http.StatusBadRequest, eventError(pb.EventResponse_INVALID_DATA),
		)
		return
	}

	deviceType := event.GetDeviceType()
	valid := false

	if deviceType == "android" {
		valid = true
	}

	if deviceType == "iphone" {
		valid = true
	}

	if !valid {
		requestError(
			ctx, w, err, "missing device type",
			http.StatusBadRequest, eventError(pb.EventResponse_INVALID_DATA),
		)
		return
	}

	identifier := event.GetEvent()

	if err := s.db.StashEventLog(identifier, deviceType); err != nil {
		requestError(
			ctx, w, err, "error saving event",
			http.StatusBadRequest, eventError(pb.EventResponse_INVALID_DATA),
		)
		return
	}

}

func (s *eventServlet) eventNonce(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != "POST" {
		log(ctx, nil).WithField("method", r.Method).Info("disallowed method")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	nonce, err := s.db.GenerateNonce()

	if err != nil {
		log(ctx, err).Error("error constructing nonce")
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	encodedNonce := b64.StdEncoding.EncodeToString(nonce)

	w.Header().Add("Access-Control-Allow-Origin", config.AppConstants.CORSAccessControlAllowOrigin)
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	if _, err := w.Write([]byte(encodedNonce + "\n")); err != nil {
		log(ctx, err).Warn("error writing nonce response")
	}
}
