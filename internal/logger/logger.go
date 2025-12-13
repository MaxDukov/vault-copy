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

// Verbose выводит сообщение только если включен verbose режим
func (l *Logger) Verbose(format string, v ...interface{}) {
	if l.verbose {
		log.Printf("[VERBOSE] "+format, v...)
	}
}

// Info выводит информационное сообщение
func (l *Logger) Info(format string, v ...interface{}) {
	log.Printf(format, v...)
}

// Error выводит сообщение об ошибке
func (l *Logger) Error(format string, v ...interface{}) {
	log.Printf("ERROR: "+format, v...)
}

// Debug выводит отладочное сообщение только если включен verbose режим
func (l *Logger) Debug(format string, v ...interface{}) {
	if l.verbose {
		log.Printf("[DEBUG] "+format, v...)
	}
}
