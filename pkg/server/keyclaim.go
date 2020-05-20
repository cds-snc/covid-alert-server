package server

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/CovidShield/server/pkg/keyclaim"
	"github.com/CovidShield/server/pkg/persistence"
	pb "github.com/CovidShield/server/pkg/proto/covidshield"

	"github.com/Shopify/goose/srvutil"
	"github.com/gorilla/mux"
	"google.golang.org/protobuf/proto"
)

func NewKeyClaimServlet(db persistence.Conn, keyClaimAuth keyclaim.Authenticator) srvutil.Servlet {
	return &keyClaimServlet{db: db, auth: keyClaimAuth}
}

type keyClaimServlet struct {
	db   persistence.Conn
	auth keyclaim.Authenticator
}

// POST /new-key-claim

func (s *keyClaimServlet) RegisterRouting(r *mux.Router) {
	r.HandleFunc("/new-key-claim", s.newKeyClaim)
	r.HandleFunc("/claim-key", s.claimKey)
}

func (s *keyClaimServlet) newKeyClaim(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method == http.MethodOptions {
		// TODO definitely do better than this for CORS
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods", "POST")
		w.Header().Add("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Referer, User-Agent")
		if _, err := w.Write([]byte("")); err != nil {
			log(ctx, err).Warn("error writing response")
		}
		return
	}

	if r.Method != "POST" {
		log(ctx, nil).WithField("method", r.Method).Info("disallowed method")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hdr := r.Header.Get("Authorization")
	region, ok := s.regionFromAuthHeader(hdr)
	if !ok {
		log(ctx, nil).WithField("header", hdr).Info("bad auth header")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	keyClaim, err := s.db.NewKeyClaim(region)
	if err != nil {
		log(ctx, err).Error("error constructing new key claim")
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	if _, err := w.Write([]byte(keyClaim + "\n")); err != nil {
		log(ctx, err).Warn("error writing response")
	}
}

func (s *keyClaimServlet) regionFromAuthHeader(header string) (string, bool) {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", false
	}
	return s.auth.Authenticate(parts[1])
}

func kcrError(errCode pb.KeyClaimResponse_ErrorCode) *pb.KeyClaimResponse {
	return &pb.KeyClaimResponse{Error: &errCode}
}

func (s *keyClaimServlet) claimKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	w.Header().Add("Content-Type", "application/x-protobuf")

	reader := http.MaxBytesReader(w, r.Body, 256)
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		requestError(
			ctx, w, err, "error reading request",
			http.StatusBadRequest, kcrError(pb.KeyClaimResponse_UNKNOWN),
		)
		return
	}

	var req pb.KeyClaimRequest
	if err := proto.Unmarshal(data, &req); err != nil {
		requestError(
			ctx, w, err, "error unmarshalling request",
			http.StatusBadRequest, kcrError(pb.KeyClaimResponse_UNKNOWN),
		)
		return
	}

	oneTimeCode := req.GetOneTimeCode()
	appPublicKey := req.GetAppPublicKey()

	serverPub, err := s.db.ClaimKey(oneTimeCode, appPublicKey)
	if err == persistence.ErrInvalidKeyFormat {
		requestError(
			ctx, w, err, "invalid key format",
			http.StatusBadRequest, kcrError(pb.KeyClaimResponse_INVALID_KEY),
		)
		return
	} else if err == persistence.ErrDuplicateKey {
		requestError(
			ctx, w, err, "duplicate key",
			http.StatusUnauthorized, kcrError(pb.KeyClaimResponse_INVALID_KEY),
		)
		return
	} else if err == persistence.ErrInvalidOneTimeCode {
		requestError(
			ctx, w, err, "invalid one time code",
			http.StatusUnauthorized, kcrError(pb.KeyClaimResponse_INVALID_ONE_TIME_CODE),
		)
		return
	} else if err != nil {
		requestError(
			ctx, w, err, "failure to claim key using OneTimeCode",
			http.StatusInternalServerError, kcrError(pb.KeyClaimResponse_SERVER_ERROR),
		)
		return
	}

	resp := &pb.KeyClaimResponse{ServerPublicKey: serverPub}

	data, err = proto.Marshal(resp)
	if err != nil {
		requestError(
			ctx, w, err, "failure to marshal response",
			http.StatusInternalServerError, kcrError(pb.KeyClaimResponse_SERVER_ERROR),
		)
		return
	}

	if _, err := w.Write(data); err != nil {
		log(ctx, err).Info("error writing response")
	}
}
