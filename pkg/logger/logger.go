// Package logger provides structured logging with file output to .logseed
package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kiridharan/seedcli/pkg/config"
	"github.com/kiridharan/seedcli/pkg/core"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// Logger implements the core.Logger interface
type Logger struct {
	mu          sync.Mutex
	level       core.LogLevel
	output      io.Writer
	fileOutput  io.Writer
	fields      []core.LogField
	useJSON     bool
	showTime    bool
	useColor    bool
	sessionID   string
}

// logEntry represents a structured log entry
type logEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	SessionID string                 `json:"session_id,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// New creates a new logger with defaults
func New() *Logger {
	return &Logger{
		level:     core.LogLevelInfo,
		output:    os.Stdout,
		useJSON:   false,
		showTime:  true,
		useColor:  true,
		sessionID: generateSessionID(),
	}
}

// NewFromConfig creates a logger from configuration
func NewFromConfig(cfg *config.Config) (*Logger, error) {
	l := New()

	// Set level
	switch cfg.Logging.Level {
	case "debug":
		l.level = core.LogLevelDebug
	case "info":
		l.level = core.LogLevelInfo
	case "warn", "warning":
		l.level = core.LogLevelWarn
	case "error":
		l.level = core.LogLevelError
	default:
		l.level = core.LogLevelInfo
	}

	l.useJSON = cfg.Logging.JSON
	l.showTime = cfg.Logging.Timestamp

	// Setup file logging
	if cfg.Logging.ToFile {
		logPath, err := config.GetLogPath()
		if err != nil {
			return nil, fmt.Errorf("failed to get log path: %w", err)
		}

		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		l.fileOutput = file
	}

	if !cfg.Logging.ToConsole {
		l.output = io.Discard
	}

	return l, nil
}

// generateSessionID creates a unique session identifier
func generateSessionID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level core.LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetOutput sets the output writer
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}

// SetFileOutput sets the file output writer
func (l *Logger) SetFileOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fileOutput = w
}

// WithFields returns a logger with additional fields
func (l *Logger) WithFields(fields ...core.LogField) core.Logger {
	newLogger := &Logger{
		level:      l.level,
		output:     l.output,
		fileOutput: l.fileOutput,
		useJSON:    l.useJSON,
		showTime:   l.showTime,
		useColor:   l.useColor,
		sessionID:  l.sessionID,
		fields:     append(l.fields, fields...),
	}
	return newLogger
}

// WithContext returns a logger with context (for now same as WithFields)
func (l *Logger) WithContext(ctx interface{}) core.Logger {
	return l
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...core.LogField) {
	if l.level <= core.LogLevelDebug {
		l.log(core.LogLevelDebug, msg, fields...)
	}
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...core.LogField) {
	if l.level <= core.LogLevelInfo {
		l.log(core.LogLevelInfo, msg, fields...)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...core.LogField) {
	if l.level <= core.LogLevelWarn {
		l.log(core.LogLevelWarn, msg, fields...)
	}
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...core.LogField) {
	if l.level <= core.LogLevelError {
		l.log(core.LogLevelError, msg, fields...)
	}
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, fields ...core.LogField) {
	l.log(core.LogLevelFatal, msg, fields...)
	os.Exit(1)
}

// Success logs a success message (info level with green color)
func (l *Logger) Success(msg string, fields ...core.LogField) {
	l.mu.Lock()
	defer l.mu.Unlock()

	allFields := append(l.fields, fields...)
	
	if l.useJSON {
		l.logJSON("success", msg, allFields)
	} else {
		l.logText("success", msg, allFields, colorGreen)
	}
}

// log handles the actual logging
func (l *Logger) log(level core.LogLevel, msg string, fields ...core.LogField) {
	l.mu.Lock()
	defer l.mu.Unlock()

	allFields := append(l.fields, fields...)
	levelStr := levelToString(level)
	color := levelToColor(level)

	if l.useJSON {
		l.logJSON(levelStr, msg, allFields)
	} else {
		l.logText(levelStr, msg, allFields, color)
	}
}

// logJSON writes a JSON formatted log
func (l *Logger) logJSON(level, msg string, fields []core.LogField) {
	entry := logEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   msg,
		SessionID: l.sessionID,
	}

	if len(fields) > 0 {
		entry.Fields = make(map[string]interface{})
		for _, f := range fields {
			entry.Fields[f.Key] = f.Value
		}
	}

	data, _ := json.Marshal(entry)
	line := string(data) + "\n"

	if l.output != nil {
		l.output.Write([]byte(line))
	}
	if l.fileOutput != nil {
		l.fileOutput.Write([]byte(line))
	}
}

// logText writes a human-readable formatted log
func (l *Logger) logText(level, msg string, fields []core.LogField, color string) {
	var consoleOutput, fileOutput string

	// Build timestamp
	timestamp := ""
	if l.showTime {
		timestamp = time.Now().Format("15:04:05") + " "
	}

	// Build level indicator
	levelIcon := levelToIcon(level)

	// Build fields string
	fieldsStr := ""
	if len(fields) > 0 {
		for _, f := range fields {
			fieldsStr += fmt.Sprintf(" %s=%v", f.Key, f.Value)
		}
	}

	// Console output (with color)
	if l.useColor {
		consoleOutput = fmt.Sprintf("%s%s%s%s%s%s %s%s%s\n",
			colorGray, timestamp, colorReset,
			color, levelIcon, colorReset,
			msg, colorGray, fieldsStr,
		)
	} else {
		consoleOutput = fmt.Sprintf("%s%s %s%s\n", timestamp, levelIcon, msg, fieldsStr)
	}

	// File output (without color)
	fileOutput = fmt.Sprintf("%s[%s] %s%s\n",
		time.Now().Format("2006-01-02 15:04:05"), level, msg, fieldsStr)

	if l.output != nil {
		l.output.Write([]byte(consoleOutput))
	}
	if l.fileOutput != nil {
		l.fileOutput.Write([]byte(fileOutput))
	}
}

// levelToString converts log level to string
func levelToString(level core.LogLevel) string {
	switch level {
	case core.LogLevelDebug:
		return "debug"
	case core.LogLevelInfo:
		return "info"
	case core.LogLevelWarn:
		return "warn"
	case core.LogLevelError:
		return "error"
	case core.LogLevelFatal:
		return "fatal"
	default:
		return "info"
	}
}

// levelToIcon returns an icon for the log level
func levelToIcon(level string) string {
	switch level {
	case "debug":
		return "🔍"
	case "info":
		return "ℹ️ "
	case "warn":
		return "⚠️ "
	case "error":
		return "❌"
	case "fatal":
		return "💀"
	case "success":
		return "✅"
	default:
		return "•"
	}
}

// levelToColor returns ANSI color for the log level
func levelToColor(level core.LogLevel) string {
	switch level {
	case core.LogLevelDebug:
		return colorGray
	case core.LogLevelInfo:
		return colorBlue
	case core.LogLevelWarn:
		return colorYellow
	case core.LogLevelError:
		return colorRed
	case core.LogLevelFatal:
		return colorRed + colorBold
	default:
		return colorReset
	}
}

// Global logger instance
var globalLogger *Logger

// Init initializes the global logger
func Init(cfg *config.Config) error {
	var err error
	globalLogger, err = NewFromConfig(cfg)
	return err
}

// InitDefault initializes the global logger with defaults
func InitDefault() {
	globalLogger = New()
	
	// Try to set up file logging
	logPath, err := config.GetLogPath()
	if err == nil {
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			globalLogger.fileOutput = file
		}
	}
}

// Get returns the global logger
func Get() *Logger {
	if globalLogger == nil {
		InitDefault()
	}
	return globalLogger
}

// Debug logs a debug message using global logger
func Debug(msg string, fields ...core.LogField) {
	Get().Debug(msg, fields...)
}

// Info logs an info message using global logger
func Info(msg string, fields ...core.LogField) {
	Get().Info(msg, fields...)
}

// Warn logs a warning message using global logger
func Warn(msg string, fields ...core.LogField) {
	Get().Warn(msg, fields...)
}

// Error logs an error message using global logger
func Error(msg string, fields ...core.LogField) {
	Get().Error(msg, fields...)
}

// Fatal logs a fatal message and exits using global logger
func Fatal(msg string, fields ...core.LogField) {
	Get().Fatal(msg, fields...)
}

// Success logs a success message using global logger
func Success(msg string, fields ...core.LogField) {
	Get().Success(msg, fields...)
}

// WithFields returns a logger with additional fields
func WithFields(fields ...core.LogField) *Logger {
	return Get().WithFields(fields...).(*Logger)
}

// F creates a log field (convenience function)
func F(key string, value interface{}) core.LogField {
	return core.F(key, value)
}

// Close closes the log file
func Close() error {
	if globalLogger != nil && globalLogger.fileOutput != nil {
		if closer, ok := globalLogger.fileOutput.(io.Closer); ok {
			return closer.Close()
		}
	}
	return nil
}

// RotateLogs rotates old log files
func RotateLogs(maxFiles int) error {
	logDir, err := config.EnsureLogDir()
	if err != nil {
		return err
	}

	files, err := filepath.Glob(filepath.Join(logDir, "seedcli-*.log"))
	if err != nil {
		return err
	}

	if len(files) <= maxFiles {
		return nil
	}

	// Sort by modification time and remove oldest
	// For simplicity, just remove files beyond the limit
	for i := 0; i < len(files)-maxFiles; i++ {
		os.Remove(files[i])
	}

	return nil
}
