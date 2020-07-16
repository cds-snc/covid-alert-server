package server

import (
	"io/ioutil"
	"net/http"

	"github.com/CovidShield/server/pkg/persistence"
	pb "github.com/CovidShield/server/pkg/proto/covidshield"

	"github.com/Shopify/goose/srvutil"
	"github.com/gorilla/mux"
	"google.golang.org/protobuf/proto"
)

func NewEventServlet(db persistence.Conn) srvutil.Servlet {
	return &eventServlet{db: db}
}

type eventServlet struct {
	db persistence.Conn
}

func (s *eventServlet) RegisterRouting(r *mux.Router) {
	r.HandleFunc("/event", s.event)
}

func eventError(errCode pb.EventResponse_ErrorCode) *pb.EventResponse {
	return &pb.EventResponse{Error: &errCode}
}

func (s *eventServlet) event(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	w.Header().Add("Content-Type", "application/x-protobuf")

	reader := http.MaxBytesReader(w, r.Body, 1024)
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		requestError(
			ctx, w, err, "error reading request",
			http.StatusBadRequest, eventError(pb.EventResponse_INVALID_KEYS),
		)
		return
	}

	var event pb.EventRequest
	if err := proto.Unmarshal(data, &event); err != nil {
		requestError(
			ctx, w, err, "error unmarshalling request",
			http.StatusBadRequest, eventError(pb.EventResponse_INVALID_KEYS),
		)
		return
	}

	serverPub := event.ServerPublicKey
	if len(serverPub) != pb.KeyLength {
		requestError(
			ctx, w, err, "server public key was not expected length",
			http.StatusBadRequest, eventError(pb.EventResponse_INVALID_KEYS),
		)
		return
	}

	_, err = s.db.PrivForPub(serverPub)
	if err != nil {
		requestError(
			ctx, w, err, "failure to resolve client keypair",
			http.StatusUnauthorized, eventError(pb.EventResponse_INVALID_KEYS),
		)
		return
	}

	_, err = pb.IntoKey(event.AppPublicKey)
	if err != nil {
		requestError(
			ctx, w, err, "app public key key was not expected length",
			http.StatusBadRequest, eventError(pb.EventResponse_INVALID_KEYS),
		)
		return
	}

	eventType := *event.Event
	if len(eventType) > 0 {
		log(ctx, nil).WithField("userEvent", eventType).Info("user event recorded")
	}

	resp := eventError(pb.EventResponse_NONE)
	data, _ = proto.Marshal(resp)

	if _, err := w.Write(data); err != nil {
		log(ctx, err).Info("error writing response")
	}
}
