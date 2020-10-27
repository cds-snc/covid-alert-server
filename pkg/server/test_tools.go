package server

import (
	"net/http"
	"os"

	"github.com/Shopify/goose/srvutil"
	"github.com/cds-snc/covid-alert-server/pkg/keyclaim"
	"github.com/cds-snc/covid-alert-server/pkg/persistence"
	"github.com/gorilla/mux"
)

func NewAdminToolsServlet(db persistence.Conn, auth keyclaim.Authenticator) srvutil.Servlet {

	log(nil, nil).Info("registering admin servlet")
	return &adminToolsServlet{db: db, auth: auth}
}

func (a adminToolsServlet) RegisterRouting(r *mux.Router) {

	// This should never happen but I want to be extra sure this isn't in production
	if os.Getenv("ENABLE_TEST_TOOLS") != "true" {
		panic("test tools must be enabled to register these routes")
	}

	log(nil, nil).Info("registering admin routes")
	r.HandleFunc("/clear-diagnosis-keys", a.clearDiagnosisKeys)

}

type adminToolsServlet struct {
	db   persistence.Conn
	auth keyclaim.Authenticator
}

func (a *adminToolsServlet) clearDiagnosisKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != "POST" {
		log(ctx, nil).WithField("method", r.Method).Info("disallowed method")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	hdr := r.Header.Get("Authorization")
	_, _, ok := a.auth.RegionFromAuthHeader(hdr)
	if !ok {
		log(ctx, nil).WithField("header", hdr).Info("bad auth header")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := a.db.ClearDiagnosisKeys(ctx); err != nil {
		log(ctx, err).Error("unable to clear diagnosis_keys")
		http.Error(w, "unable to clear diagnosis_keys", http.StatusInternalServerError)
		return
	}

	log(ctx, nil).Info("cleared diagnosis_keys")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("cleared diagnosis_keys"))

}
