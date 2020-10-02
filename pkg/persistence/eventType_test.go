package persistence

import "testing"

func TestEventType_IsValid(t *testing.T) {
	var test EventType = "foo"
	if err := test.IsValid(); err == nil {
		t.Errorf("Invalid Event Type Passed")
	}

	for _, et := range []EventType{
		OTKClaimed,
		OTKGenerated,
		OTKExhausted,
		OTKRegenerated,
		OTKExpired,
		OTKUnclaimed,
	} {
		if err := et.IsValid(); err != nil {
			t.Errorf("Valid EventType failed: %s", et)
		}
	}
}
