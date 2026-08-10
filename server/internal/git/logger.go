package git

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel represents the severity level of a log entry
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
	LogLevelFatal LogLevel = "FATAL"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp  time.Time              `json:"timestamp"`
	Level      LogLevel               `json:"level"`
	Message    string                 `json:"message"`
	Module     string                 `json:"module,omitempty"`
	User       string                 `json:"user,omitempty"`
	Repository string                 `json:"repository,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
	Duration   time.Duration          `json:"duration,omitempty"`
	StatusCode int                    `json:"status_code,omitempty"`
	Method     string                 `json:"method,omitempty"`
	Path       string                 `json:"path,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Extra      map[string]interface{} `json:"extra,omitempty"`
}

// Logger provides structured logging capabilities
type Logger struct {
	mu      sync.Mutex
	output  io.Writer
	level   LogLevel
	module  string
	formats map[LogLevel]string
}

// NewLogger creates a new structured logger
func NewLogger(output io.Writer, level LogLevel) *Logger {
	if output == nil {
		output = os.Stdout
	}
	return &Logger{
		output:  output,
		level:   level,
		formats: make(map[LogLevel]string),
	}
}

// WithModule creates a new logger with a module name
func (l *Logger) WithModule(module string) *Logger {
	return &Logger{
		output: l.output,
		level:  l.level,
		module: module,
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// shouldLog checks if a message at the given level should be logged
func (l *Logger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		LogLevelDebug: 0,
		LogLevelInfo:  1,
		LogLevelWarn:  2,
		LogLevelError: 3,
		LogLevelFatal: 4,
	}
	return levels[level] >= levels[l.level]
}

// log writes a structured log entry
func (l *Logger) log(level LogLevel, message string, extra map[string]interface{}) {
	if !l.shouldLog(level) {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Module:    l.module,
		Extra:     extra,
	}

	l.writeEntry(entry)
}

// Debug logs a debug message
func (l *Logger) Debug(message string, extra ...map[string]interface{}) {
	var extraMap map[string]interface{}
	if len(extra) > 0 {
		extraMap = extra[0]
	}
	l.log(LogLevelDebug, message, extraMap)
}

// Info logs an info message
func (l *Logger) Info(message string, extra ...map[string]interface{}) {
	var extraMap map[string]interface{}
	if len(extra) > 0 {
		extraMap = extra[0]
	}
	l.log(LogLevelInfo, message, extraMap)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, extra ...map[string]interface{}) {
	var extraMap map[string]interface{}
	if len(extra) > 0 {
		extraMap = extra[0]
	}
	l.log(LogLevelWarn, message, extraMap)
}

// Error logs an error message
func (l *Logger) Error(message string, err error, extra ...map[string]interface{}) {
	extraMap := make(map[string]interface{})
	if len(extra) > 0 && extra[0] != nil {
		extraMap = extra[0]
	}
	if err != nil {
		extraMap["error"] = err.Error()
	}
	l.log(LogLevelError, message, extraMap)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(message string, err error, extra ...map[string]interface{}) {
	extraMap := make(map[string]interface{})
	if len(extra) > 0 && extra[0] != nil {
		extraMap = extra[0]
	}
	if err != nil {
		extraMap["error"] = err.Error()
	}
	l.log(LogLevelFatal, message, extraMap)
	os.Exit(1)
}

// LogRequest logs an HTTP request
func (l *Logger) LogRequest(method, path string, statusCode int, duration time.Duration, user string, extra ...map[string]interface{}) {
	extraMap := make(map[string]interface{})
	if len(extra) > 0 && extra[0] != nil {
		extraMap = extra[0]
	}

	entry := LogEntry{
		Timestamp:  time.Now(),
		Level:      LogLevelInfo,
		Message:    "HTTP request",
		Module:     l.module,
		Method:     method,
		Path:       path,
		StatusCode: statusCode,
		Duration:   duration,
		User:       user,
		Extra:      extraMap,
	}

	l.writeEntry(entry)
}

// LogError logs an error with context
func (l *Logger) LogError(message string, err error, user string, extra ...map[string]interface{}) {
	extraMap := make(map[string]interface{})
	if len(extra) > 0 {
		extraMap = extra[0]
	}
	if err != nil {
		extraMap["error"] = err.Error()
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     LogLevelError,
		Message:   message,
		Module:    l.module,
		User:      user,
		Extra:     extraMap,
	}

	l.writeEntry(entry)
}

// writeEntry writes a log entry to the output
func (l *Logger) writeEntry(entry LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(l.output, `{"timestamp":"%s","level":"ERROR","message":"Failed to marshal log entry","error":"%s"}`,
			time.Now().Format(time.RFC3339), err.Error())
		return
	}

	fmt.Fprintln(l.output, string(data))
}

// Global logger instance
var defaultLogger = NewLogger(os.Stdout, LogLevelInfo)

// GetLogger returns the global logger
func GetLogger() *Logger {
	return defaultLogger
}

// SetLogLevel sets the global log level
func SetLogLevel(level LogLevel) {
	defaultLogger.SetLevel(level)
}

// InitLogger initializes the logger with file output
func InitLogger(logDir string, level LogLevel) error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	logFile := filepath.Join(logDir, fmt.Sprintf("iforge-%s.log", time.Now().Format("2006-01-02")))
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Write to both stdout and file
	multiWriter := io.MultiWriter(os.Stdout, file)
	defaultLogger = NewLogger(multiWriter, level)

	return nil
}
