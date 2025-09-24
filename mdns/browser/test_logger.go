package browser

import (
	"fmt"
	"testing"
	"time"
)

// testLogger is a logger that writes to a testing.T.
type testLogger struct {
	t *testing.T
}

// NewTestLogger creates a new logger that writes to the given testing.T.
func NewTestLogger(t *testing.T) Logger {
	return &testLogger{t: t}
}

func (l *testLogger) log(level, format string, args ...interface{}) {
	l.t.Logf(fmt.Sprintf("%s %s: %s", time.Now().Format(time.RFC3339Nano), level, format), args...)
}

func (l *testLogger) Debugf(format string, args ...interface{}) { l.log("DEBUG", format, args...) }

func (l *testLogger) Infof(format string, args ...interface{}) { l.log("INFO", format, args...) }

func (l *testLogger) Warningf(format string, args ...interface{}) {
	l.log("WARN", format, args...)
}

func (l *testLogger) Errorf(format string, args ...interface{}) { l.log("ERROR", format, args...) }
