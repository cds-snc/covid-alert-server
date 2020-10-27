package server

import (
	"net/http"
	"os"

	"github.com/Shopify/goose/srvutil"
	"github.com/cds-snc/covid-alert-server/pkg/keyclaim"
	"github.com/cds-snc/covid-alert-server/pkg/persistence"
	"github.com/gorilla/mux"
)

func NewTestToolsServlet(db persistence.Conn, auth keyclaim.Authenticator) srvutil.Servlet {

	log(nil, nil).Info("registering admin servlet")
	return &testToolsServlet{db: db, auth: auth}
}

func (t testToolsServlet) RegisterRouting(r *mux.Router) {

	// This should never happen but I want to be extra sure this isn't in production
	if os.Getenv("ENABLE_TEST_TOOLS") != "true" {
		panic("test tools must be enabled to register these routes")
	}

	if os.Getenv("ENV") == "production" {
		panic("attempting to enable test tools in production")
	}
	log(nil, nil).Info("registering admin routes")
	r.HandleFunc("/clear-diagnosis-keys", t.clearDiagnosisKeys)

}

type testToolsServlet struct {
	db   persistence.Conn
	auth keyclaim.Authenticator
}

func (t *testToolsServlet) clearDiagnosisKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != "POST" {
		log(ctx, nil).WithField("method", r.Method).Info("disallowed method")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	hdr := r.Header.Get("Authorization")
	_, _, ok := t.auth.RegionFromAuthHeader(hdr)
	if !ok {
		log(ctx, nil).WithField("header", hdr).Info("bad auth header")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := t.db.ClearDiagnosisKeys(ctx); err != nil {
		log(ctx, err).Error("unable to clear diagnosis_keys")
		http.Error(w, "unable to clear diagnosis_keys", http.StatusInternalServerError)
		return
	}

	log(ctx, nil).Info("cleared diagnosis_keys")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("cleared diagnosis_keys"))

}
