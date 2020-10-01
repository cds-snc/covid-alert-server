package persistence

import "fmt"

// EventType the type of the event that happened
type EventType string

// OTKClaimed One Time Key Claimed
// OTKGenerated One Time Key Generated
// OTKRegenerated One Time Key Regenerated
// OTKExpired One Time Key Expired
// OTKExhausted One Time Key exhausted all it's TEKs
const (
	OTKClaimed   EventType = "OTKClaimed"
	OTKUnclaimed EventType = "OTKUnclaimed"
	OTKGenerated EventType = "OTKGenerated"
	OTKExpired   EventType = "OTKExpired"
	OTKExhausted		 EventType = "OTKExhausted"
	OTKRegenerated EventType ="OTKRegenerated"
)

// IsValid validates the Event Type against a list of allowed strings
func (et EventType) IsValid() error {
	switch et {
	case OTKGenerated, OTKClaimed, OTKExpired, OTKRegenerated, OTKExhausted, OTKUnclaimed:
		return nil
	}
	return fmt.Errorf("invalid EventType: (%s)", et)
}

