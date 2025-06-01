package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/2bleere/voice-ferry/pkg/config"
)

// LogLevel represents log levels
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// LogFormat represents log output formats
type LogFormat string

const (
	FormatJSON LogFormat = "json"
	FormatText LogFormat = "text"
)

// ContextKey represents context keys for logging
type ContextKey string

const (
	CallIDKey    ContextKey = "call_id"
	SessionIDKey ContextKey = "session_id"
	DialogIDKey  ContextKey = "dialog_id"
	SourceIPKey  ContextKey = "source_ip"
	UserAgentKey ContextKey = "user_agent"
	ComponentKey ContextKey = "component"
	OperationKey ContextKey = "operation"
	RequestIDKey ContextKey = "request_id"
)

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	*slog.Logger
	component string
}

// NewLogger creates a new logger based on configuration
func NewLogger(cfg config.LoggingConfig, component string) (*Logger, error) {
	// Determine log level
	var level slog.Level
	switch LogLevel(cfg.Level) {
	case LevelDebug:
		level = slog.LevelDebug
	case LevelInfo:
		level = slog.LevelInfo
	case LevelWarn:
		level = slog.LevelWarn
	case LevelError:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create handler options
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.IncludeSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize timestamp format
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   "timestamp",
					Value: slog.StringValue(a.Value.Time().Format(time.RFC3339Nano)),
				}
			}
			return a
		},
	}

	// Determine output destination
	var writer io.Writer = os.Stdout
	if cfg.File != "" {
		// Ensure directory exists
		dir := filepath.Dir(cfg.File)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}

		file, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}
		writer = file
	}

	// Create handler based on format
	var handler slog.Handler
	switch LogFormat(cfg.Format) {
	case FormatJSON:
		handler = slog.NewJSONHandler(writer, opts)
	case FormatText:
		handler = slog.NewTextHandler(writer, opts)
	default:
		handler = slog.NewJSONHandler(writer, opts)
	}

	// Create base logger
	baseLogger := slog.New(handler)

	// Add default fields
	logger := baseLogger.With(
		"component", component,
		"version", cfg.Version,
		"instance_id", cfg.InstanceID,
		"operation", "init",
	)

	return &Logger{
		Logger:    logger,
		component: component,
	}, nil
}

// WithContext returns a logger with context values
func (l *Logger) WithContext(ctx context.Context) *Logger {
	logger := l.Logger

	// Extract context values and add to logger
	if callID := ctx.Value(CallIDKey); callID != nil {
		logger = logger.With("call_id", callID)
	}
	if sessionID := ctx.Value(SessionIDKey); sessionID != nil {
		logger = logger.With("session_id", sessionID)
	}
	if dialogID := ctx.Value(DialogIDKey); dialogID != nil {
		logger = logger.With("dialog_id", dialogID)
	}
	if sourceIP := ctx.Value(SourceIPKey); sourceIP != nil {
		logger = logger.With("source_ip", sourceIP)
	}
	if userAgent := ctx.Value(UserAgentKey); userAgent != nil {
		logger = logger.With("user_agent", userAgent)
	}
	if operation := ctx.Value(OperationKey); operation != nil {
		logger = logger.With("operation", operation)
	}
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		logger = logger.With("request_id", requestID)
	}

	return &Logger{
		Logger:    logger,
		component: l.component,
	}
}

// WithFields returns a logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}

	return &Logger{
		Logger:    l.Logger.With(args...),
		component: l.component,
	}
}

// SIPRequestLogger logs SIP request details
func (l *Logger) SIPRequestLogger(method, callID, fromURI, toURI, sourceIP string) *Logger {
	return &Logger{
		Logger: l.Logger.With(
			"sip_method", method,
			"call_id", callID,
			"from_uri", fromURI,
			"to_uri", toURI,
			"source_ip", sourceIP,
		),
		component: l.component,
	}
}

// SIPResponseLogger logs SIP response details
func (l *Logger) SIPResponseLogger(statusCode int, reasonPhrase, callID string) *Logger {
	return &Logger{
		Logger: l.Logger.With(
			"sip_status_code", statusCode,
			"sip_reason_phrase", reasonPhrase,
			"call_id", callID,
		),
		component: l.component,
	}
}

// RoutingLogger logs routing decision details
func (l *Logger) RoutingLogger(ruleID, action, sourceIP, requestURI string) *Logger {
	return &Logger{
		Logger: l.Logger.With(
			"routing_rule_id", ruleID,
			"routing_action", action,
			"source_ip", sourceIP,
			"request_uri", requestURI,
		),
		component: l.component,
	}
}

// MediaLogger logs media session details
func (l *Logger) MediaLogger(callID, sessionID, rtpengineInstance string) *Logger {
	return &Logger{
		Logger: l.Logger.With(
			"call_id", callID,
			"session_id", sessionID,
			"rtpengine_instance", rtpengineInstance,
		),
		component: l.component,
	}
}

// DatabaseLogger logs database operation details
func (l *Logger) DatabaseLogger(operation, database, table string, duration time.Duration) *Logger {
	return &Logger{
		Logger: l.Logger.With(
			"db_operation", operation,
			"database", database,
			"table", table,
			"duration_ms", duration.Milliseconds(),
		),
		component: l.component,
	}
}

// APILogger logs API request details
func (l *Logger) APILogger(method, path, clientIP, userAgent string, statusCode int, duration time.Duration) *Logger {
	return &Logger{
		Logger: l.Logger.With(
			"http_method", method,
			"http_path", path,
			"client_ip", clientIP,
			"user_agent", userAgent,
			"http_status_code", statusCode,
			"duration_ms", duration.Milliseconds(),
		),
		component: l.component,
	}
}

// SecurityLogger logs security-related events
func (l *Logger) SecurityLogger(event, sourceIP, userID string) *Logger {
	return &Logger{
		Logger: l.Logger.With(
			"security_event", event,
			"source_ip", sourceIP,
			"user_id", userID,
		),
		component: l.component,
	}
}

// ErrorLogger logs error details with stack trace if available
func (l *Logger) ErrorLogger(err error, operation string) *Logger {
	logger := l.Logger.With(
		"error", err.Error(),
		"operation", operation,
	)

	// Add stack trace if it's a structured error type
	// This would depend on your error handling strategy

	return &Logger{
		Logger:    logger,
		component: l.component,
	}
}

// PerformanceLogger logs performance metrics
func (l *Logger) PerformanceLogger(operation string, duration time.Duration, success bool) *Logger {
	return &Logger{
		Logger: l.Logger.With(
			"performance_operation", operation,
			"duration_ms", duration.Milliseconds(),
			"success", success,
		),
		component: l.component,
	}
}

// AuditLogger logs audit events
func (l *Logger) AuditLogger(action, resource, userID, sourceIP string, success bool) *Logger {
	return &Logger{
		Logger: l.Logger.With(
			"audit_action", action,
			"audit_resource", resource,
			"user_id", userID,
			"source_ip", sourceIP,
			"success", success,
			"timestamp", time.Now().UTC(),
		),
		component: l.component,
	}
}

// ContextWithCallID adds call ID to context
func ContextWithCallID(ctx context.Context, callID string) context.Context {
	return context.WithValue(ctx, CallIDKey, callID)
}

// ContextWithSessionID adds session ID to context
func ContextWithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, SessionIDKey, sessionID)
}

// ContextWithDialogID adds dialog ID to context
func ContextWithDialogID(ctx context.Context, dialogID string) context.Context {
	return context.WithValue(ctx, DialogIDKey, dialogID)
}

// ContextWithSourceIP adds source IP to context
func ContextWithSourceIP(ctx context.Context, sourceIP string) context.Context {
	return context.WithValue(ctx, SourceIPKey, sourceIP)
}

// ContextWithUserAgent adds user agent to context
func ContextWithUserAgent(ctx context.Context, userAgent string) context.Context {
	return context.WithValue(ctx, UserAgentKey, userAgent)
}

// ContextWithComponent adds component to context
func ContextWithComponent(ctx context.Context, component string) context.Context {
	return context.WithValue(ctx, ComponentKey, component)
}

// ContextWithOperation adds operation to context
func ContextWithOperation(ctx context.Context, operation string) context.Context {
	return context.WithValue(ctx, OperationKey, operation)
}

// ContextWithRequestID adds request ID to context
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// LoggerManager manages multiple loggers for different components
type LoggerManager struct {
	loggers map[string]*Logger
	config  config.LoggingConfig
}

// NewLoggerManager creates a new logger manager
func NewLoggerManager(cfg config.LoggingConfig) *LoggerManager {
	return &LoggerManager{
		loggers: make(map[string]*Logger),
		config:  cfg,
	}
}

// GetLogger returns a logger for a specific component
func (lm *LoggerManager) GetLogger(component string) (*Logger, error) {
	if logger, exists := lm.loggers[component]; exists {
		return logger, nil
	}

	logger, err := NewLogger(lm.config, component)
	if err != nil {
		return nil, err
	}

	lm.loggers[component] = logger
	return logger, nil
}

// GetOrCreateLogger returns an existing logger or creates a new one
func (lm *LoggerManager) GetOrCreateLogger(component string) *Logger {
	logger, err := lm.GetLogger(component)
	if err != nil {
		// Fallback to basic logger if creation fails
		return &Logger{
			Logger:    slog.Default(),
			component: component,
		}
	}
	return logger
}

// Close closes all file handles (if any)
func (lm *LoggerManager) Close() error {
	// Implementation would depend on whether we're tracking file handles
	// For now, this is a placeholder
	return nil
}

// SIP returns a logger with operation=sip
func (l *Logger) SIP() *Logger {
	return l.WithFields(map[string]interface{}{"operation": "sip"})
}

// Routing returns a logger with operation=routing
func (l *Logger) Routing() *Logger {
	return l.WithFields(map[string]interface{}{"operation": "routing"})
}

// Media returns a logger with operation=media
func (l *Logger) Media() *Logger {
	return l.WithFields(map[string]interface{}{"operation": "media"})
}

// Info logs a message at info level.
func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

// Debug logs a message at debug level.
func (l *Logger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

// Warn logs a message at warn level.
func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

// Error logs a message at error level.
func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}

// parseLogLevel parses a string log level to slog.Level
func parseLogLevel(level string) (slog.Level, error) {
	switch LogLevel(strings.ToLower(level)) {
	case LevelDebug:
		return slog.LevelDebug, nil
	case LevelInfo:
		return slog.LevelInfo, nil
	case LevelWarn:
		return slog.LevelWarn, nil
	case LevelError:
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("invalid log level: %s", level)
	}
}

// createHandler creates a slog.Handler based on format
func createHandler(format string, writer io.Writer, level slog.Level, addSource bool) slog.Handler {
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: addSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   "timestamp",
					Value: slog.StringValue(a.Value.Time().Format(time.RFC3339Nano)),
				}
			}
			return a
		},
	}
	switch strings.ToLower(format) {
	case "json":
		return slog.NewJSONHandler(writer, opts)
	case "text":
		return slog.NewTextHandler(writer, opts)
	default:
		return slog.NewTextHandler(writer, opts)
	}
}
