package persistence

import(
	"database/sql"
	"fmt"
	"math"
	"time"
)


// Writes

// OtkDuration duration of time an OTK is live sorted by originator
// Originator where the OTK was generated
// Duration the number of hours that the otk was live
type OtkDuration struct {
	Originator string
	Duration time.Duration
}

func saveOtkDuration(tx *sql.Tx, otkDuration OtkDuration) error {

	hoursLive := otkDuration.Duration.Hours()
	hours := math.Ceil(hoursLive)

	originator := translateToken(otkDuration.Originator)

	if _, err := tx.Exec(`
		INSERT INTO otk_life_duration
		(originator, hours, date, count)
		VALUES(?, ?, ?, 1) ON DUPLICATE KEY UPDATE count = count + 1`,
		originator, hours, time.Now().Format("2006-01-02")); err != nil{
		return err
	}
	return nil
}


// Reads

// AggregateOtkDuration the aggregate of OTK Lifetimes to the nearest integer (CEIL(duration))
// Source the originator of the OTK associated with this event
// Date	the date the OTK was claimed on
// Hours the number of hours live, calculated by taking time live and running it through a ceil function
// Count the number of otks that lived for this many hours
type AggregateOtkDuration struct {
	Source      string `json:"source"`
	Date        string `json:"date"`
	Hours       int    `json:"hours"`
	Count       int64  `json:"count"`
}

func (c *conn) GetAggregateOtkDurationsByDate(date string) ([]AggregateOtkDuration, error) {
	return getAggregateOtkDurationsByDate(c.db, date)
}

func  getAggregateOtkDurationsByDate(db *sql.DB, date string) ([]AggregateOtkDuration, error) {

	if date == "" {
		return nil, fmt.Errorf("a date is required for querying events")
	}

	rows, err := db.Query(`
	SELECT originator, hours, date, count 
	FROM otk_life_duration
	WHERE otk_life_duration.date = ?`, date)

	if err != nil {
		return nil, err
	}

	var durations []AggregateOtkDuration

	for rows.Next() {
		u := AggregateOtkDuration{}
		var t time.Time

		err := rows.Scan(&u.Source, &u.Hours, &t, &u.Count)

		if err != nil {
			return nil, err
		}

		u.Date = t.Format("2006-01-02")
		durations = append(durations, u)
	}

	if durations == nil {
		durations = make([]AggregateOtkDuration, 0)
	}
	return durations, nil
}


