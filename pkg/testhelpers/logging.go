package testhelpers

import (
	"github.com/Shopify/goose/logger"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func SetupTestLogging(log *logger.Logger) (*test.Hook, *logger.Logger) {
	// Capture logs
	oldLog := log

	nullLog, hook := test.NewNullLogger()
	nullLog.ExitFunc = func(code int) {}

	*log = func(ctx logger.Valuer, err ...error) *logrus.Entry {
		return logrus.NewEntry(nullLog)
	}
	return hook, oldLog
}

func AssertLog(t *testing.T, hook *test.Hook, length int, level logrus.Level, msg string) {
	assert.Equal(t, length, len(hook.Entries))
	assert.Equal(t, level, hook.LastEntry().Level)
	assert.Equal(t, msg, hook.LastEntry().Message)
	hook.Reset()
}
