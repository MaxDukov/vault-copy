package logger

import (
	"log"
	"vault-copy/internal/config"
)

type Logger struct {
	verbose bool
}

func NewLogger(cfg *config.Config) *Logger {
	return &Logger{
		verbose: cfg.Verbose,
	}
}

// Verbose prints a message only if verbose mode is enabled
func (l *Logger) Verbose(format string, v ...interface{}) {
	if l.verbose {
		log.Printf("[VERBOSE] "+format, v...)
	}
}

// Info prints an informational message
func (l *Logger) Info(format string, v ...interface{}) {
	log.Printf(format, v...)
}

// Error prints an error message
func (l *Logger) Error(format string, v ...interface{}) {
	log.Printf("ERROR: "+format, v...)
}

// Debug prints a debug message only if verbose mode is enabled
func (l *Logger) Debug(format string, v ...interface{}) {
	if l.verbose {
		log.Printf("[DEBUG] "+format, v...)
	}
}
