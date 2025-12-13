package mocks

import (
	"testing"
)

type TestLogger struct {
	t *testing.T
}

func NewTestLogger(t *testing.T) *TestLogger {
	return &TestLogger{t: t}
}

func (l *TestLogger) Verbose(format string, v ...interface{}) {
	l.t.Logf("[VERBOSE] "+format, v...)
}

func (l *TestLogger) Info(format string, v ...interface{}) {
	l.t.Logf("[INFO] "+format, v...)
}

func (l *TestLogger) Error(format string, v ...interface{}) {
	l.t.Logf("[ERROR] "+format, v...)
}

func (l *TestLogger) Debug(format string, v ...interface{}) {
	l.t.Logf("[DEBUG] "+format, v...)
}

func (l *TestLogger) Fatal(format string, v ...interface{}) {
	l.t.Fatalf(format, v...)
}
