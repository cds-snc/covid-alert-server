package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/cds-snc/covid-alert-server/pkg/config"
	"github.com/cds-snc/covid-alert-server/pkg/keyclaim"
	"github.com/cds-snc/covid-alert-server/pkg/persistence"
	pb "github.com/cds-snc/covid-alert-server/pkg/proto/covidshield"

	"github.com/Shopify/goose/srvutil"
	"github.com/golang/protobuf/ptypes"
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
	r.HandleFunc("/new-key-claim/{hashID:[0-9,a-z]{128}}", s.newKeyClaim)
	r.HandleFunc("/claim-key", s.claimKeyWrapper)
}

func (s *keyClaimServlet) newKeyClaim(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	if r.Method == http.MethodOptions {
		w.Header().Add("Access-Control-Allow-Origin", config.AppConstants.CORSAccessControlAllowOrigin)
		w.Header().Add("Access-Control-Allow-Methods", "POST")
		w.Header().Add("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Referer, User-Agent")
		if _, err := w.Write([]byte("")); err != nil {
			log(ctx, err).Warn("error writing response")
		}
		return
	}

	if r.Method != "POST" {
		log(ctx, nil).WithField("method", r.Method).Info("disallowed method")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	hdr := r.Header.Get("Authorization")
	region, originator, ok := s.regionFromAuthHeader(hdr)
	if !ok {
		log(ctx, nil).WithField("header", hdr).Info("bad auth header")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	/*
		Since the region that is being retrieved above is no longer the MCC code for canada
		we want to override that with the actual value. The reason for this is that we are now
		using the bearer token environment variable to store the name of the provice it's associated with

		Also please note this hurts me to not go through the rest of the code to pull out the region code
		I will open an issue to continue with this work.
	*/
	region = config.AppConstants.RegionCode

	hashID := vars["hashID"]

	keyClaim, err := s.db.NewKeyClaim(region, originator, hashID)
	if err == persistence.ErrHashIDClaimed {
		log(ctx, err).Info("hashID used")
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	} else if err != nil {
		log(ctx, err).Error("error constructing new key claim")
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	event := persistence.Event{
		Originator: originator,
		DeviceType: persistence.Server,
		Identifier: persistence.OTKGenerated,
		Date:  time.Now(),
		Count: 1,
	}
	if err := s.db.SaveEvent(event); err != nil {
		// We don't necessarily want to crash if we were unable to log a metric
		log(nil, err).Warn(fmt.Sprintf("Unable to log event: %#v\n", event))
	}

	w.Header().Add("Access-Control-Allow-Origin", config.AppConstants.CORSAccessControlAllowOrigin)
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	if _, err := w.Write([]byte(keyClaim + "\n")); err != nil {
		log(ctx, err).Warn("error writing response")
	}
}

func (s *keyClaimServlet) regionFromAuthHeader(header string) (string, string, bool) {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", "", false
	}
	region, ok := s.auth.Authenticate(parts[1])
	return region, parts[1], ok
}

func kcrError(errCode pb.KeyClaimResponse_ErrorCode, triesRemaining int) *pb.KeyClaimResponse {
	tr := uint32(triesRemaining)
	return &pb.KeyClaimResponse{Error: &errCode, TriesRemaining: &tr}
}

func (s *keyClaimServlet) claimKeyWrapper(w http.ResponseWriter, r *http.Request) {
	_ = s.claimKey(w, r)
}

func (s *keyClaimServlet) claimKey(w http.ResponseWriter, r *http.Request) result {
	ctx := r.Context()

	// be extremely careful not to log this or otherwise cause it to be persisted
	// other than transiently in the failed attempts table.
	ip := getIP(r)

	triesRemaining, banDuration, err := s.db.CheckClaimKeyBan(ip)
	if err != nil {
		kcre := kcrError(pb.KeyClaimResponse_SERVER_ERROR, triesRemaining)
		return requestError(ctx, w, err, "database error checking claim-key ban", http.StatusInternalServerError, kcre)
	} else if triesRemaining == 0 {
		kcre := kcrError(pb.KeyClaimResponse_TEMPORARY_BAN, triesRemaining)
		kcre.RemainingBanDuration = ptypes.DurationProto(banDuration)
		return requestError(ctx, w, err, "error reading request", http.StatusTooManyRequests, kcre)
	}

	w.Header().Add("Content-Type", "application/x-protobuf")

	reader := http.MaxBytesReader(w, r.Body, 256)
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return requestError(
			ctx, w, err, "error reading request",
			http.StatusBadRequest, kcrError(pb.KeyClaimResponse_UNKNOWN, triesRemaining),
		)
	}

	var req pb.KeyClaimRequest
	if err := proto.Unmarshal(data, &req); err != nil {
		return requestError(
			ctx, w, err, "error unmarshalling request",
			http.StatusBadRequest, kcrError(pb.KeyClaimResponse_UNKNOWN, triesRemaining),
		)
	}

	oneTimeCode := req.GetOneTimeCode()

	// Handle odd app inputs
	oneTimeCode = strings.ReplaceAll(oneTimeCode, " ", "")
	oneTimeCode = strings.ReplaceAll(oneTimeCode, "-", "")

	appPublicKey := req.GetAppPublicKey()

	serverPub, err := s.db.ClaimKey(oneTimeCode, appPublicKey)
	if err == persistence.ErrInvalidKeyFormat {
		return requestError(
			ctx, w, err, "invalid key format",
			http.StatusBadRequest, kcrError(pb.KeyClaimResponse_INVALID_KEY, triesRemaining),
		)
	} else if err == persistence.ErrDuplicateKey {
		return requestError(
			ctx, w, err, "duplicate key",
			http.StatusUnauthorized, kcrError(pb.KeyClaimResponse_INVALID_KEY, triesRemaining),
		)
	} else if err == persistence.ErrInvalidOneTimeCode {
		triesRemaining, banDuration, err := s.db.ClaimKeyFailure(ip)
		if err != nil {
			kcre := kcrError(pb.KeyClaimResponse_SERVER_ERROR, triesRemaining)
			msg := "database error recording claim-key failure"
			return requestError(ctx, w, err, msg, http.StatusInternalServerError, kcre)
		}
		kcre := kcrError(pb.KeyClaimResponse_INVALID_ONE_TIME_CODE, triesRemaining)
		kcre.RemainingBanDuration = ptypes.DurationProto(banDuration)
		return requestError(ctx, w, err, "invalid one time code", http.StatusUnauthorized, kcre)
	} else if err != nil {
		return requestError(
			ctx, w, err, "failure to claim key using OneTimeCode",
			http.StatusInternalServerError, kcrError(pb.KeyClaimResponse_SERVER_ERROR, triesRemaining),
		)
	}

	maxTries := uint32(config.AppConstants.MaxConsecutiveClaimKeyFailures)
	resp := &pb.KeyClaimResponse{ServerPublicKey: serverPub, TriesRemaining: &maxTries}

	data, err = proto.Marshal(resp)
	if err != nil {
		return requestError(
			ctx, w, err, "failure to marshal response",
			http.StatusInternalServerError, kcrError(pb.KeyClaimResponse_SERVER_ERROR, triesRemaining),
		)
	}

	if _, err := w.Write(data); err != nil {
		log(ctx, err).Info("error writing response")
	}

	if err := s.db.ClaimKeySuccess(ip); err != nil {
		log(ctx, err).Warn("error recording claim-key success")
	}

	return result{}
}

var numeric = regexp.MustCompile("^[0-9]+$")

func getIP(r *http.Request) string {
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	if forwarded != "" {
		IPList := strings.Split(forwarded, ",")
		return strings.TrimSpace(IPList[len(IPList)-1])
	}
	// If the RemoteAddr is of the form $ip:$port, return only the IP
	parts := strings.Split(r.RemoteAddr, ":")
	if len(parts) == 2 && numeric.MatchString(parts[1]) {
		return parts[0]
	}
	return r.RemoteAddr
}
