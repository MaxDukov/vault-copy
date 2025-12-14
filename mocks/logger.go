package mocks

import (
	"log"
	"os"
)

// Logger is a mock logger for testing
type Logger struct {
	logger *log.Logger
}

// NewLogger creates a new mock logger
func NewLogger() *Logger {
	return &Logger{
		logger: log.New(os.Stdout, "", 0),
	}
}

// Info logs an info message
func (l *Logger) Info(format string, v ...interface{}) {
	l.logger.Printf("[INFO] "+format, v...)
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	l.logger.Printf("[ERROR] "+format, v...)
}

// Verbose logs a verbose message
func (l *Logger) Verbose(format string, v ...interface{}) {
	l.logger.Printf("[VERBOSE] "+format, v...)
}
