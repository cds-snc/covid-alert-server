package persistence

import "testing"

func TestDeviceType_IsValid(t *testing.T) {
	var test DeviceType = "foo"

	if err := test.IsValid(); err == nil {
		t.Errorf("Invalid Device Type Passed")
	}

	for _,dt := range [3]DeviceType{Server, Android, iOS} {
		if err := dt.IsValid(); err != nil {
			t.Errorf("Valid Device Type Failed")
		}
	}
}

func TestEventType_IsValid(t *testing.T) {
	var test EventType = "foo"
	if err := test.IsValid(); err == nil {
		t.Errorf("Invalid Event Type Passed")
	}

	for _,et := range [2]EventType{OTKClaimed, OTKGenerated}{
		if err := et.IsValid(); err != nil {
			t.Errorf("Valid EventType failed: %s",et)
		}
	}
}