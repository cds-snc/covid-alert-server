package server

import (
	"context"
	"io/ioutil"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/CovidShield/server/pkg/persistence"
	pb "github.com/CovidShield/server/pkg/proto/covidshield"

	"github.com/Shopify/goose/srvutil"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/nacl/box"
	"google.golang.org/protobuf/proto"
)

func NewUploadServlet(db persistence.Conn) srvutil.Servlet {
	return &uploadServlet{db: db}
}

type uploadServlet struct {
	db persistence.Conn
}

func (s *uploadServlet) RegisterRouting(r *mux.Router) {
	r.HandleFunc("/upload", s.upload)
}

func uploadError(errCode pb.EncryptedUploadResponse_ErrorCode) *pb.EncryptedUploadResponse {
	return &pb.EncryptedUploadResponse{Error: &errCode}
}

func (s *uploadServlet) upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	w.Header().Add("Content-Type", "application/x-protobuf")

	reader := http.MaxBytesReader(w, r.Body, 1024)
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		requestError(
			ctx, w, err, "error reading request",
			http.StatusBadRequest, uploadError(pb.EncryptedUploadResponse_UNKNOWN),
		)
		return
	}

	var seu pb.EncryptedUploadRequest
	if err := proto.Unmarshal(data, &seu); err != nil {
		requestError(
			ctx, w, err, "error unmarshalling request",
			http.StatusBadRequest, uploadError(pb.EncryptedUploadResponse_UNKNOWN),
		)
		return
	}

	serverPub := seu.ServerPublicKey
	if len(serverPub) != pb.KeyLength {
		requestError(
			ctx, w, err, "server public key was not expected length",
			http.StatusBadRequest, uploadError(pb.EncryptedUploadResponse_INVALID_CRYPTO_PARAMETERS),
		)
		return
	}

	serverPriv, err := s.db.PrivForPub(serverPub)
	if err != nil {
		requestError(
			ctx, w, err, "failure to resolve client keypair",
			http.StatusUnauthorized, uploadError(pb.EncryptedUploadResponse_INVALID_KEYPAIR),
		)
		return
	}

	nonce, err := pb.IntoNonce(seu.Nonce)
	if err != nil {
		requestError(
			ctx, w, err, "nonce was not expected length",
			http.StatusBadRequest, uploadError(pb.EncryptedUploadResponse_INVALID_CRYPTO_PARAMETERS),
		)
		return
	}

	appPubKey, err := pb.IntoKey(seu.AppPublicKey)
	if err != nil {
		requestError(
			ctx, w, err, "app public key key was not expected length",
			http.StatusBadRequest, uploadError(pb.EncryptedUploadResponse_INVALID_CRYPTO_PARAMETERS),
		)
		return
	}

	privKey, err := pb.IntoKey(serverPriv)
	if err != nil {
		requestError(
			ctx, w, err, "server private key was not expected length",
			http.StatusInternalServerError, uploadError(pb.EncryptedUploadResponse_SERVER_ERROR),
		)
		return
	}

	// decrypt payload
	plaintext, ok := box.Open(nil, seu.Payload, nonce, appPubKey, privKey)
	if !ok {
		requestError(
			ctx, w, nil, "failure to decrypt payload",
			http.StatusBadRequest, uploadError(pb.EncryptedUploadResponse_DECRYPTION_FAILED),
		)
		return
	}

	// unmarshall into Upload
	var upload pb.Upload
	if err := proto.Unmarshal(plaintext, &upload); err != nil {
		requestError(
			ctx, w, err, "error unmarshalling request payload",
			http.StatusBadRequest, uploadError(pb.EncryptedUploadResponse_INVALID_PAYLOAD),
		)
		return
	}

	if len(upload.GetKeys()) == 0 {
		requestError(
			ctx, w, err, "no keys provided",
			http.StatusBadRequest, uploadError(pb.EncryptedUploadResponse_NO_KEYS_IN_PAYLOAD),
		)
		return
	}

	if len(upload.GetKeys()) > pb.MaxKeysInUpload {
		requestError(
			ctx, w, err, "too many keys provided",
			http.StatusBadRequest, uploadError(pb.EncryptedUploadResponse_TOO_MANY_KEYS),
		)
		return
	}

	ts := time.Unix(upload.GetTimestamp().Seconds, 0)
	if math.Abs(time.Since(ts).Seconds()) > 3600 {
		requestError(
			ctx, w, err, "invalid timestamp",
			http.StatusBadRequest, uploadError(pb.EncryptedUploadResponse_INVALID_TIMESTAMP),
		)
		return
	}

	if ok := validateKeys(ctx, w, upload.GetKeys()); !ok {
		return // requestError done by validateKeys
	}

	err = s.db.StoreKeys(appPubKey, upload.GetKeys())
	if err == persistence.ErrKeyConsumed {
		requestError(
			ctx, w, err, "key is used up",
			http.StatusBadRequest, uploadError(pb.EncryptedUploadResponse_INVALID_KEYPAIR),
		)
		return
	} else if err != nil {
		requestError(
			ctx, w, err, "failed to store diagnosis keys",
			http.StatusInternalServerError, uploadError(pb.EncryptedUploadResponse_SERVER_ERROR),
		)
		return
	}

	resp := uploadError(pb.EncryptedUploadResponse_NONE)
	data, err = proto.Marshal(resp)
	if err != nil {
		requestError(
			ctx, w, err, "error marshalling response",
			http.StatusInternalServerError, uploadError(pb.EncryptedUploadResponse_SERVER_ERROR),
		)
		return
	}

	if _, err := w.Write(data); err != nil {
		log(ctx, err).Info("error writing response")
	}
}

func validateKey(ctx context.Context, w http.ResponseWriter, key *pb.TemporaryExposureKey) bool {
	if key.GetRollingPeriod() != 144 {
		requestError(
			ctx, w, nil, "missing or invalid rollingPeriod",
			http.StatusBadRequest, uploadError(pb.EncryptedUploadResponse_INVALID_ROLLING_PERIOD),
		)
		return false
	}

	if len(key.GetKeyData()) != 16 {
		requestError(
			ctx, w, nil, "invalid key data",
			http.StatusBadRequest, uploadError(pb.EncryptedUploadResponse_INVALID_KEY_DATA),
		)
		return false
	}

	if key.GetRollingStartIntervalNumber() == 0 {
		requestError(
			ctx, w, nil, "invalid rolling start number",
			http.StatusBadRequest, uploadError(pb.EncryptedUploadResponse_INVALID_ROLLING_START_INTERVAL_NUMBER),
		)
		return false
	}

	level := key.GetTransmissionRiskLevel()
	if level < 0 || level > 8 {
		requestError(
			ctx, w, nil, "invalid transmission risk level",
			http.StatusBadRequest, uploadError(pb.EncryptedUploadResponse_INVALID_TRANSMISSION_RISK_LEVEL),
		)
		return false
	}

	return true
}

func validateKeys(ctx context.Context, w http.ResponseWriter, keys []*pb.TemporaryExposureKey) bool {
	for _, key := range keys {
		if ok := validateKey(ctx, w, key); !ok {
			return false
		}
	}

	var ints []int
	for _, key := range keys {
		rsin := int(key.GetRollingStartIntervalNumber())
		ints = append(ints, rsin)
	}

	sort.Ints(ints)

	min := ints[0]
	max := ints[len(ints)-1]
	maxEnd := max + 144

	if maxEnd-min > (144 * 14) {
		requestError(
			ctx, w, nil, "sequence of rollingStartIntervalNumbers exceeds 14 days",
			http.StatusBadRequest, uploadError(pb.EncryptedUploadResponse_INVALID_ROLLING_START_INTERVAL_NUMBER),
		)
		return false
	}

	lastEnd := 0
	for _, rsn := range ints {
		if rsn < lastEnd {
			requestError(
				ctx, w, nil, "overlapping or duplicate rollingStartIntervalNumbers",
				http.StatusBadRequest, uploadError(pb.EncryptedUploadResponse_INVALID_ROLLING_START_INTERVAL_NUMBER),
			)
			return false
		}
		lastEnd = rsn + 144
	}

	return true
}
