package server

import (
	b64 "encoding/base64"
	"net/http"

	"github.com/CovidShield/server/pkg/config"
	"github.com/CovidShield/server/pkg/persistence"
	pb "github.com/CovidShield/server/pkg/proto/covidshield"

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

}

func (s *eventServlet) eventNonce(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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
