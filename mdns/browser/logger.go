package browser

import (
	"log"
)

// Logger defines a standard logging interface that can be implemented by
// various logging libraries, such as the CoreDNS logger or the standard log package.
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warningf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// DefaultLogger is a simple implementation of the Logger interface that uses
// the standard Go `log` package.
type DefaultLogger struct{}

// NewDefaultLogger creates a new DefaultLogger.
func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{}
}

// Debugf logs a debug message.
func (l *DefaultLogger) Debugf(format string, args ...interface{}) {
	log.Printf("DEBUG: "+format, args...)
}

// Infof logs an info message.
func (l *DefaultLogger) Infof(format string, args ...interface{}) {
	log.Printf("INFO: "+format, args...)
}

// Warningf logs a warning message.
func (l *DefaultLogger) Warningf(format string, args ...interface{}) {
	log.Printf("WARN: "+format, args...)
}

// Errorf logs an error message.
func (l *DefaultLogger) Errorf(format string, args ...interface{}) {
	log.Printf("ERROR: "+format, args...)
}
