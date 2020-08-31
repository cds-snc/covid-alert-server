package server

import (
	"net/http"

	"github.com/CovidShield/server/pkg/config"
	"github.com/CovidShield/server/pkg/persistence"

	"github.com/Shopify/goose/srvutil"
	"github.com/gorilla/mux"
)

func NewEventServlet(cache persistence.RedisConn) srvutil.Servlet {
	return &eventServlet{cache: cache}
}

type eventServlet struct {
	cache persistence.RedisConn
}

func (s *eventServlet) RegisterRouting(r *mux.Router) {
	r.HandleFunc("/event/nonce", s.eventNonce)
}

func (s *eventServlet) eventNonce(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != "POST" {
		log(ctx, nil).WithField("method", r.Method).Info("disallowed method")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	nonce, err := s.cache.GenerateNonce()

	if err != nil {
		log(ctx, err).Error("error constructing nonce")
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Access-Control-Allow-Origin", config.AppConstants.CORSAccessControlAllowOrigin)
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	if _, err := w.Write([]byte(nonce + "\n")); err != nil {
		log(ctx, err).Warn("error writing nonce response")
	}
}
