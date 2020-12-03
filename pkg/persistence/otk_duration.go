package persistence

import(
	"database/sql"
	"math"
	"time"
)

// OtkDuration duration of time an OTK is live sorted by originator
// Originator where the OTK was generated
// Duration the number of hours that the otk was live
type OtkDuration struct {
	Originator string
	Duration time.Duration
}

func saveOtkDuration(db *sql.DB, otkDuration OtkDuration) error {

	hoursLive := otkDuration.Duration.Hours()
	hours := math.Ceil(hoursLive)

	originator := translateToken(otkDuration.Originator)

	if _, err := db.Exec(`
		INSERT INTO otk_life_duration
		(originator, hours, date, count)
		VALUES(?, ?, ?, 1) ON DUPLICATE KEY UPDATE count = count + 1`,
		originator, hours, time.Now().Format("2006-01-02")); err != nil{
		return err
	}
	return nil
}

