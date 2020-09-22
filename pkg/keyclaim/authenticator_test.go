package keyclaim

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/cds-snc/covid-alert-server/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestNewAuthenticator(t *testing.T) {

	// Init config
	config.InitConfig()

	os.Setenv("KEY_CLAIM_TOKEN", "")
	assert.PanicsWithValue(t, "no KEY_CLAIM_TOKEN", func() { NewAuthenticator() }, "KEY_CLAIM_TOKEN needs to be defined")

	os.Setenv("KEY_CLAIM_TOKEN", "foobaz")
	assert.PanicsWithValue(t, "invalid KEY_CLAIM_TOKEN", func() { NewAuthenticator() }, "KEY_CLAIM_TOKEN must include a `=` and have "+fmt.Sprint(config.AppConstants.AssignmentParts)+"parts")

	os.Setenv("KEY_CLAIM_TOKEN", strings.Repeat("a", 64)+"=302")
	assert.PanicsWithValue(t, "token too long", func() { NewAuthenticator() }, "KEY_CLAIM_TOKEN must include secret that is at less than 64 characters long")

	os.Setenv("KEY_CLAIM_TOKEN", strings.Repeat("a", 19)+"=302")
	assert.PanicsWithValue(t, "token too short", func() { NewAuthenticator() }, "KEY_CLAIM_TOKEN must include secret that is at least 20 characters long")

	os.Setenv("KEY_CLAIM_TOKEN", strings.Repeat("a", 20)+"="+strings.Repeat("a", 32))
	assert.PanicsWithValue(t, "region too long", func() { NewAuthenticator() }, "KEY_CLAIM_TOKEN must include a region that is less than 32 characters long")

	tokens := make(map[string]string)
	tokens[strings.Repeat("a", 20)] = "302"
	tokens[strings.Repeat("b", 20)] = "302"
	expected := &authenticator{tokens: tokens}

	os.Setenv("KEY_CLAIM_TOKEN", strings.Repeat("a", 20)+"=302:"+strings.Repeat("b", 20)+"=302")
	assert.Equal(t, expected, NewAuthenticator(), "Returns an authenticator struct with a map of valid tokens and regions")
}

func TestAuthenticate(t *testing.T) {

	// Initialise Authenticator object
	os.Setenv("KEY_CLAIM_TOKEN", strings.Repeat("a", 20)+"=302:"+strings.Repeat("b", 20)+"=302")
	authenticator := NewAuthenticator()

	// Valid token
	expectedRegion := "302"
	expectedBool := true
	receivedRegion, receivedBool := authenticator.Authenticate(strings.Repeat("a", 20))
	assert.Equal(t, expectedRegion, receivedRegion, "Expected region is 302 on valid")
	assert.Equal(t, expectedBool, receivedBool, "Expected bool is true on invalid token")

	// Invalid token
	expectedRegion = ""
	expectedBool = false
	receivedRegion, receivedBool = authenticator.Authenticate(strings.Repeat("c", 20))
	assert.Equal(t, expectedRegion, receivedRegion, "Expected region is nil on invalid token")
	assert.Equal(t, expectedBool, receivedBool, "Expected bool is false on invalid token")
}
