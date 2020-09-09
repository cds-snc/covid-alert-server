package keyclaim

import (
	"os"
	"strings"

	"github.com/CovidShield/server/pkg/config"
)

type Authenticator interface {
	Authenticate(string) (string, bool)
}

type authenticator struct {
	tokens map[string]string
}

// 1234deadbeefcafe=1:c0ffeec0ffeec0ffee=2
// These are two keys with region IDs 1 and 2 respectively. Keys should be much
// longer than this but still hexadecimal.
func NewAuthenticator() Authenticator {
	authTokens := make(map[string]string)
	tokens := os.Getenv("KEY_CLAIM_TOKEN")
	if tokens == "" {
		panic("no KEY_CLAIM_TOKEN")
	}
	for _, tokenWithProvince := range strings.Split(tokens, ":") {
		assignmentParts := config.AppConstants.AssignmentParts
		parts := strings.SplitN(tokenWithProvince, "=", assignmentParts)


		if len(parts) != assignmentParts {
			panic("invalid KEY_CLAIM_TOKEN")
		}

		token := parts[0]
		if len(token) > 63 {
			panic("token too long")
		}
		if len(token) < 20 {
			panic("token too short")
		}

		province := parts[1]
		if len(province) > 31 {
			panic("province too long")
		}

		authTokens[token] = province
	}

	return &authenticator{tokens: authTokens}
}

func (a *authenticator) Authenticate(token string) (string, bool) {
	province, ok := a.tokens[token]
	return province, ok
}
