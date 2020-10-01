package persistence

import (
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
)



func Test_translateToken(t *testing.T) {

	token3 := strings.Repeat("c",20)

	originator := translateToken(token1)
	assert.Equal(t, onApi, originator)

	originator = translateTokenForLogs(token2)
	assert.Equal(t, "b...b", originator)

	originator = translateTokenForLogs(token3)
	assert.Equal(t, "c...c", originator)

}

func Test_translateTokenForLogs(t *testing.T) {

	token3 := strings.Repeat("c",20)

	originator := translateTokenForLogs(token1)
	assert.Equal(t, onApi, originator)

	originator = translateToken(token2)
	assert.Equal(t, token2, originator)

	originator = translateToken(token3)
	assert.Equal(t, token3, originator)

}

func setupSaveEventMock(mock sqlmock.Sqlmock, event Event){
	mock.ExpectBegin()
	mock.ExpectExec(
		`INSERT INTO events
		(source, identifier, device_type, date, count)
		VALUES (?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE count = count + ?`).WithArgs(
		onApi,
		event.Identifier,
		event.Date,
		event.DeviceType,
		event.Date,
		event.Count,
	)
	mock.ExpectRollback()
}

func Test_SaveEvent(t *testing.T) {

	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()

	event := Event {
		Identifier: OTKGenerated,
		Originator: token1,
		Count: 1,
		DeviceType: Server,
		Date: time.Now(),
	}

	setupSaveEventMock(mock ,event)

	saveEvent(db, event)

}


func Test_LogEvent(t *testing.T){

	hook := test.NewGlobal()

	now := time.Now()
	event := Event {
		Identifier: OTKGenerated,
		Originator: token1,
		Count: 1,
		DeviceType: Server,
		Date: now,
	}

	LogEvent(nil, nil, event)

	assert.Equal(t,logrus.WarnLevel, hook.LastEntry().Level)
	assert.Contains(t, hook.LastEntry().Message, "Unable to log event")
	assert.Equal(t, 1,  hook.LastEntry().Data["Count"] )
	assert.Equal(t, onApi, hook.LastEntry().Data["Originator"])
	assert.Equal(t, OTKGenerated, hook.LastEntry().Data["Identifier"])
	assert.Equal(t, Server, hook.LastEntry().Data["DeviceType"])
	assert.Equal(t, now, hook.LastEntry().Data["Date"])

	event = Event {
		Originator: token2,
	}

	// Test anonymizing bearer tokens
	LogEvent(nil, nil, event)

	assert.Equal(t, "b...b", hook.LastEntry().Data["Originator"])
}