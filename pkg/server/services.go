package server

import (
	"crypto/rand"
	"encoding/json"
	"math/big"
	"net/http"
	"os/exec"
	"strings"

	"github.com/Shopify/goose/srvutil"
	"github.com/gorilla/mux"
)

var branch string
var revision string

func NewServicesServlet() srvutil.Servlet {
	s := &servicesServlet{}
	return srvutil.PrefixServlet(s, "/services")
}

type servicesServlet struct{}
type version struct {
	Branch   string `json:"branch"`
	Revision string `json:"revision"`
}

func (s *servicesServlet) RegisterRouting(r *mux.Router) {
	r.HandleFunc("/ping", s.ping)
	r.HandleFunc("/version.json", s.version)
	r.HandleFunc("/urandom.bin", s.urandom)
	r.HandleFunc("/sample", s.sample)
}

func (s *servicesServlet) ping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	w.Header().Add("Cache-Control", "no-store")
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	if _, err := w.Write([]byte("OK\n")); err != nil {
		log(ctx, err).Info("error writing response")
	}
}

func (s *servicesServlet) version(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	w.Header().Add("Cache-Control", "no-store")
	w.Header().Add("Content-Type", "application/json; charset=utf-8")

	version := version{
		Branch:   branch,
		Revision: revision,
	}

	js, err := json.Marshal(version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(js); err != nil {
		log(ctx, err).Info("error writing response")
	}
}

func (s *servicesServlet) urandom(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	cmd, _ := exec.Command("head", "-c1000000", "/dev/urandom").Output()

	w.Header().Add("Cache-Control", "application/octet-stream")
	if _, err := w.Write([]byte(cmd)); err != nil {
		log(ctx, err).Info("error writing response")
	}
}

func (s *servicesServlet) sample(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var code []string

	for i := 0; i < 10000; i++ {
		oneTimeCode, _ := generateOneTimeCode()
		code = append(code, oneTimeCode)
	}

	w.Header().Add("Cache-Control", "no-store")
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	if _, err := w.Write([]byte(strings.Join(code, "\n"))); err != nil {
		log(ctx, err).Info("error writing response")
	}
}

func generateOneTimeCode() (string, error) {
	characterSets := [2][]rune{
		[]rune("AEFHJKLQRSUWXYZ"),
		[]rune("2456789"),
	}

	characterSetLength := int64(len(characterSets))

	seg1, err := rand.Int(rand.Reader, big.NewInt(characterSetLength))
	seg2, err := rand.Int(rand.Reader, big.NewInt(characterSetLength))
	seg3, err := rand.Int(rand.Reader, big.NewInt(characterSetLength))

	oneTimeCode := genRandom(characterSets[seg1.Int64()], 3) +
		genRandom(characterSets[seg2.Int64()], 3) +
		genRandom(characterSets[seg3.Int64()], 4)

	return oneTimeCode, err
}

// Generates a string of random characters based on a
// passed list of characters and a desired length. For each
// position in the desired length, generates a random number
// between 0 and the length of the character set.
func genRandom(chars []rune, length int64) string {
	var b strings.Builder
	for i := int64(0); i < length; i++ {
		nBig, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b.WriteRune(chars[nBig.Int64()])
	}
	return b.String()
}
