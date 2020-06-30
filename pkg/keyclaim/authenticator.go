package keyclaim

import (
	"os"
	"strings"
)

type Authenticator interface {
	Authenticate(string) (string, bool)
}

type authenticator struct {
	tokens map[string]string
}

const assignmentParts = 2

// 1234deadbeefcafe=1:c0ffeec0ffeec0ffee=2
// These are two keys with region IDs 1 and 2 respectively. Keys should be much
// longer than this but still hexadecimal.
func NewAuthenticator() Authenticator {
	authTokens := make(map[string]string)
	tokens := os.Getenv("KEY_CLAIM_TOKEN")
	if tokens == "" {
		panic("no KEY_CLAIM_TOKEN")
	}
	for _, tokenWithRegion := range strings.Split(tokens, ":") {
		parts := strings.SplitN(tokenWithRegion, "=", assignmentParts)
		if len(parts) != assignmentParts {
			panic("invalid KEY_CLAIM_TOKEN")
		}
		if len(parts[0]) > 63 {
			panic("token too long")
		}
		if len(parts[0]) < 20 {
			panic("token too short")
		}
		if len(parts[1]) > 31 {
			panic("region too long")
		}
		authTokens[parts[0]] = parts[1]
	}

	return &authenticator{tokens: authTokens}
}

func (a *authenticator) Authenticate(token string) (string, bool) {
	region, ok := a.tokens[token]
	return region, ok
}
