package server

import (
	"testing"

	keyclaim "github.com/CovidShield/server/mocks/pkg/keyclaim"
	persistence "github.com/CovidShield/server/mocks/pkg/persistence"

	"github.com/stretchr/testify/assert"
)

func TestNewKeyClaimServlet(t *testing.T) {
	db := &persistence.Conn{}
	auth := &keyclaim.Authenticator{}

	expected := &keyClaimServlet{
		db:   db,
		auth: auth,
	}
	assert.Equal(t, expected, NewKeyClaimServlet(db, auth), "should return a new keyClaimServlet struct")
}

func TestRegisterRoutingKeyClaim(t *testing.T) {
	servlet := NewKeyClaimServlet(&persistence.Conn{}, &keyclaim.Authenticator{})
	router := Router()
	servlet.RegisterRouting(router)

	expectedPaths := GetPaths(router)
	assert.Contains(t, expectedPaths, "/new-key-claim", "should include a /new-key-claim path")
	assert.Contains(t, expectedPaths, "/new-key-claim/{hashID:[0-9,a-z]{128}}", "should include a /new-key-claim/{hashID:[0-9,a-z]{128}} path")
	assert.Contains(t, expectedPaths, "/claim-key", "should include a claim-key path")
}

func TestNewKeyClaim(t *testing.T) {
}

func TestRegionFromAuthHeader(t *testing.T) {
	s := NewKeyClaimServlet(&persistence.Conn{}, &keyclaim.Authenticator{})

	goodHeader := "Bearer thisisaverylongtoken"

	// Header needs to have two words
	receivedRegion, receivedToken, receivedResult := *s.regionFromAuthHeader("ooo")
	assert.Equals(t, receivedRegion, "", "Region should be blank")
	assert.Equals(t, receivedToken, "", "Token should be blank")
	assert.Equals(t, receivedResult, false, "Bool should be false")
}
