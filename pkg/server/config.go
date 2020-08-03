package server

import (
	"fmt"
	"net/http"

	"github.com/Shopify/goose/srvutil"
	"github.com/gorilla/mux"
)

const (
	response = `{"minimumRiskScore":28,"attenuationLevelValues":[1,1,8,8,8,8,8,8],"attenuationWeight":50,"daysSinceLastExposureLevelValues":[0,1,1,1,1,1,1,1],"daysSinceLastExposureWeight":50,"durationLevelValues":[0,0,0,0,4,4,5,8],"durationWeight":50,"transmissionRiskLevelValues":[1,1,1,1,1,1,1,1],"transmissionRiskWeight":50}`
)

func NewConfigServlet() srvutil.Servlet {
	return &configServlet{}
}

type configServlet struct{}

func (s *configServlet) RegisterRouting(r *mux.Router) {
	r.HandleFunc("/exposure-configuration/{region:[\\w]+}.json", s.exposureConfig)
}

func (s *configServlet) exposureConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	fmt.Fprintf(w, response)
}
