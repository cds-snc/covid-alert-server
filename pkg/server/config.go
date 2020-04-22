package server

import (
	"fmt"
	"net/http"

	"github.com/Shopify/goose/srvutil"
	"github.com/gorilla/mux"
)

const (
	response = `{"minimumRiskScore":0,"attenuationLevelValues":[1,2,3,4,5,6,7,8],"attenuationWeight":50,"daysSinceLastExposureLevelValues":[1,2,3,4,5,6,7,8],"daysSinceLastExposureWeight":50,"durationLevelValues":[1,2,3,4,5,6,7,8],"durationWeight":50,"transmissionRiskLevelValues":[1,2,3,4,5,6,7,8],"transmissionRiskWeight":50}`
)

func NewConfigServlet() srvutil.Servlet {
	return &configServlet{}
}

type configServlet struct{}

func (s *configServlet) RegisterRouting(r *mux.Router) {
	r.HandleFunc("/exposure-configuration/{region:[\\w]+}.json", s.exposureConfig)
}

// TODO: serve this from somewhere else
func (s *configServlet) exposureConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	fmt.Fprintf(w, response)
}
