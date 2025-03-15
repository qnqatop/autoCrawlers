package logger

import (
	"fmt"
	"log/slog"
	"os"
)

// Logger is a struct for embedding std loggers.
type Logger struct {
	d, e SimpleLogger
}

func NewLogger(verbose bool) Logger {
	// Default error logger with stderr
	l := Logger{
		e: NewSimpleLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelError,
		}))),
	}

	if verbose {
		l.d = NewSimpleLogger(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
		})))
	}
	return l
}

// Printf prints message to Stdout (app.log variable) if a.verbose is set.
func (l Logger) Printf(format string, args ...interface{}) { l.d.Printf(format, args...) }

// Errorf prints message to Stderr (l.warn variable).
func (l Logger) Errorf(format string, args ...interface{}) { l.e.Printf(format, args...) }

// With adds key-value context to logger and returns new copy of logger object.
// All new prints would be with defined context before.
func (l Logger) With(key string, value interface{}) Logger {
	l.d = l.d.With(key, value)
	l.e = l.e.With(key, value)
	return l
}

func (l Logger) Write(b []byte) (int, error) {
	return l.d.Write(b)
}

// SimpleLogger is minimal instance of logger object. Most of the time you should use Logger.
type SimpleLogger struct {
	lg     *slog.Logger
	prefix string
}

func NewSimpleLogger(core *slog.Logger) SimpleLogger {
	return SimpleLogger{lg: core}
}

// Printf prints message to logger's output.
func (l SimpleLogger) Printf(format string, args ...interface{}) {
	// go tool vet help printf hint
	if false {
		_ = fmt.Sprintf(format, args...) // enable printf checking
	}

	if l.lg == nil {
		return
	}
	msg := fmt.Sprintf(l.prefix+format, args...)
	l.lg.Info(msg) // or l.lg.Error(msg) depending on context
}

// With adds key-value context to logger and returns new copy of logger object.
// All new prints would be with defined context before.
func (l SimpleLogger) With(key string, value interface{}) SimpleLogger {
	l.prefix += fmt.Sprintf("%s=%v ", key, value)
	return l
}

func (l SimpleLogger) Write(b []byte) (int, error) {
	if l.lg == nil {
		return 0, nil
	}
	l.lg.Info(string(b))
	return len(b), nil
}
