package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/cds-snc/covid-alert-server/pkg/keyclaim"
)

// Event the event that we are to log
// Identifier The EventType of the event
// DeviceType the DeviceType of the event
// Date The date the event was generated on
// Count The number of times the event occurred
// Originator The bearerToken that the event belongs to
type Event struct {
	Identifier EventType
	DeviceType DeviceType
	Date       time.Time
	Count      int
	Originator string
}

var originatorLookup keyclaim.Authenticator

// SetupLookup Setup the originator lookup used to map events to bearerTokens
func SetupLookup(lookup keyclaim.Authenticator) {
	originatorLookup = lookup
}

func translateToken(token string) string {
	region, ok := originatorLookup.Authenticate(token)

	// If we forgot to map a token to a PT just return the token
	if region == "302" {
		return token
	}

	// If it's an old token or unknown just return the token
	if ok == false {
		return token
	}

	return region
}

// translateTokenForLogs Since we don't want to log bearer tokens to the log file we only use the first and last character
func translateTokenForLogs(token string) string {
	region, ok := originatorLookup.Authenticate(token)

	if region == "302" || ok == false {
		return fmt.Sprintf("%s...%s", token[0:1], token[len(token)-1:])
	}

	return region
}

// LogEvent Log a failed Event
func LogEvent(ctx context.Context, err error, event Event) {

	log(ctx, err).WithFields(logrus.Fields{
		"Originator": translateTokenForLogs(event.Originator),
		"DeviceType": event.DeviceType,
		"Identifier": event.Identifier,
		"Date":       event.Date,
		"Count":      event.Count,
	}).Warn("Unable to log event")
}

// SaveEvent log an Event in the database
func (c *conn) SaveEvent(event Event) error {

	if err := saveEvent(c.db, event); err != nil {
		return err
	}
	return nil
}

func saveEvent(db *sql.DB, e Event) error {
	if err := e.DeviceType.IsValid(); err != nil {
		return err
	}

	if err := e.Identifier.IsValid(); err != nil {
		return err
	}

	originator := translateToken(e.Originator)

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`
		INSERT INTO events
		(source, identifier, device_type, date, count)
		VALUES (?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE count = count + ?`,
		originator, e.Identifier, e.DeviceType, e.Date.Format("2006-01-02"), e.Count, e.Count); err != nil {

		if err := tx.Rollback(); err != nil {
			return err
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// Events the aggregate of events identified in Identifier by Source
// Source the bearer token that generated these events
// Date the date the events occurs
// Count the number of times this event occurred
// Identifier the event that occurred
type Events struct {
	Source     string `json:"source"`
	Date       string `json:"date"`
	Count      int64  `json:"count"`
	Identifier string `json:"identifier"`
}

// GetServerEvents get all the events that occurred in a day
func (c *conn) GetServerEvents(date string) ([]Events, error) {
	return getServerEventsByType(c.db, date)
}

func getServerEventsByType(db *sql.DB, date string) ([]Events, error) {

	if date == "" {
		return nil, fmt.Errorf("a date is required for querying events")
	}

	rows, err := db.Query(`
	SELECT identifier, source, date, count 
	FROM events 
	WHERE events.device_type = ? AND events.date = ?`,
		Server, date)

	if err != nil {
		return nil, err
	}

	var events []Events

	for rows.Next() {
		e := Events{}
		var t time.Time

		err := rows.Scan(&e.Identifier, &e.Source, &t, &e.Count)

		if err != nil {
			return nil, err
		}

		e.Date = t.Format("2006-01-02")
		events = append(events, e)
	}

	if events == nil {
		events = make([]Events, 0)
	}
	return events, nil
}

// Uploads the aggregate of uploads identified in orignator by Source
// Source the bearer token that generated these uploads
// Date the date the upload occurs
// Count the number of keys uploaded
// FirstUpload if this was the first upload by a user
type Uploads struct {
	Source      string `json:"source"`
	Date        string `json:"date"`
	Count       int64  `json:"count"`
	FirstUpload bool   `json:"first_upload"`
}

// GetTEKUploads get all the events that occurred in a day
func (c *conn) GetTEKUploads(date string) ([]Uploads, error) {
	return getTEKUploadsByDay(c.db, date)
}

func getTEKUploadsByDay(db *sql.DB, date string) ([]Uploads, error) {

	if date == "" {
		return nil, fmt.Errorf("a date is required for querying events")
	}

	rows, err := db.Query(`
	SELECT originator, date, count, first_upload 
	FROM tek_upload_count 
	WHERE tek_upload_count.date = ?`, date)

	if err != nil {
		return nil, err
	}

	var uploads []Uploads

	for rows.Next() {
		u := Uploads{}
		var t time.Time

		err := rows.Scan(&u.Source, &t, &u.Count, &u.FirstUpload)

		if err != nil {
			return nil, err
		}

		u.Date = t.Format("2006-01-02")
		uploads = append(uploads, u)
	}

	if uploads == nil {
		uploads = make([]Uploads, 0)
	}
	return uploads, nil
}
