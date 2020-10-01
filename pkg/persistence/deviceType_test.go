package persistence

import "testing"

func TestDeviceType_IsValid(t *testing.T) {
	var test DeviceType = "foo"

	if err := test.IsValid(); err == nil {
		t.Errorf("Invalid Device Type Passed")
	}

	for _,dt := range []DeviceType{
		Server,
		Android,
		IOS,
	} {
		if err := dt.IsValid(); err != nil {
			t.Errorf("Valid Device Type Failed")
		}
	}
}
