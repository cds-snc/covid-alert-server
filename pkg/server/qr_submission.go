package server

import (
	"io/ioutil"
	"net/http"

	"github.com/Shopify/goose/srvutil"
	"github.com/cds-snc/covid-alert-server/pkg/keyclaim"
	"github.com/cds-snc/covid-alert-server/pkg/persistence"
	pb "github.com/cds-snc/covid-alert-server/pkg/proto/covidshield"
	"github.com/gorilla/mux"
	"google.golang.org/protobuf/proto"
)

type OutbreakEventServlet struct {
	db   persistence.Conn
	auth keyclaim.Authenticator
}

func NewOutbreakEventServlet(db persistence.Conn, OutbreakEventAuth keyclaim.Authenticator) srvutil.Servlet {
	s := &OutbreakEventServlet{db: db, auth: OutbreakEventAuth}

	return srvutil.PrefixServlet(s, "/qr")
}

func (s *OutbreakEventServlet) RegisterRouting(r *mux.Router) {
	r.HandleFunc("/new-event", s.newExposureEvent)
}

func qrUploadResponse(errCode pb.OutbreakEventResponse_ErrorCode) *pb.OutbreakEventResponse {
	return &pb.OutbreakEventResponse{Error: &errCode}
}

func (s *OutbreakEventServlet) newExposureEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != "POST" {
		log(ctx, nil).WithField("method", r.Method).Info("disallowed method")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	hdr := r.Header.Get("Authorization")
	_, originator, ok := s.auth.RegionFromAuthHeader(hdr)
	if !ok {
		log(ctx, nil).WithField("header", hdr).Info("bad auth header")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Add("Content-Type", "application/x-protobuf")

	reader := http.MaxBytesReader(w, r.Body, 1024)
	data, err := ioutil.ReadAll(reader)

	if err != nil {
		requestError(
			ctx, w, err, "error reading request",
			http.StatusBadRequest, qrUploadResponse(pb.OutbreakEventResponse_UNKNOWN),
		)
		return
	}

	var submission pb.OutbreakEvent
	if err := proto.Unmarshal(data, &submission); err != nil {
		requestError(
			ctx, w, err, "error unmarshalling request",
			http.StatusBadRequest, qrUploadResponse(pb.OutbreakEventResponse_UNKNOWN),
		)
		return
	}

	if len(submission.GetLocationId()) != 36 {
		requestError(
			ctx, w, err, "Location ID is not valid",
			http.StatusBadRequest, qrUploadResponse(pb.OutbreakEventResponse_INVALID_ID),
		)
		return
	}

	if submission.GetStartTime().Seconds < 1 || submission.GetEndTime().Seconds < 1 {
		requestError(
			ctx, w, err, "missing/invalid timestamp",
			http.StatusBadRequest, qrUploadResponse(pb.OutbreakEventResponse_MISSING_TIMESTAMP),
		)
		return
	}

	if submission.GetEndTime().Seconds-submission.GetStartTime().Seconds < 1 {
		requestError(
			ctx, w, err, "invalid timeperiod",
			http.StatusBadRequest, qrUploadResponse(pb.OutbreakEventResponse_PERIOD_INVALID),
		)
		return
	}

	// Save the new QR Submission
	err = s.db.NewOutbreakEvent(ctx, originator, &submission)

	if err != nil {
		requestError(
			ctx, w, err, "error saving QR submission",
			http.StatusBadRequest, qrUploadResponse(pb.OutbreakEventResponse_SERVER_ERROR),
		)
		return
	}

	resp := qrUploadResponse(pb.OutbreakEventResponse_NONE)
	data, err = proto.Marshal(resp)
	if err != nil {
		requestError(
			ctx, w, err, "error marshalling response",
			http.StatusInternalServerError, qrUploadResponse(pb.OutbreakEventResponse_SERVER_ERROR),
		)
		return
	}

	if _, err := w.Write(data); err != nil {
		log(ctx, err).Info("error writing response")
	}

}
