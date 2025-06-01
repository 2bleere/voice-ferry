package logging

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2bleere/voice-ferry/pkg/config"
)

func TestNewLogger_Development(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:       "debug",
		Format:      "text",
		Development: true,
	}

	logger, err := NewLogger(cfg, "test-component")
	require.NoError(t, err)
	require.NotNil(t, logger)

	assert.Equal(t, "test-component", logger.component)
	assert.NotNil(t, logger.Logger)
}

func TestNewLogger_Production(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:       "info",
		Format:      "json",
		Development: false,
	}

	logger, err := NewLogger(cfg, "prod-component")
	require.NoError(t, err)
	require.NotNil(t, logger)

	assert.Equal(t, "prod-component", logger.component)
	assert.NotNil(t, logger.Logger)
}

func TestNewLogger_InvalidLevel(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:  "invalid",
		Format: "text",
	}

	logger, err := NewLogger(cfg, "test")
	assert.Error(t, err)
	assert.Nil(t, logger)
	assert.Contains(t, err.Error(), "invalid log level")
}

func TestNewLogger_InvalidFormat(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:  "info",
		Format: "invalid",
	}

	logger, err := NewLogger(cfg, "test")
	assert.Error(t, err)
	assert.Nil(t, logger)
	assert.Contains(t, err.Error(), "invalid log format")
}

func TestLogger_WithContext(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:       "debug",
		Format:      "text",
		Development: true,
	}

	logger, err := NewLogger(cfg, "test")
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), CallIDKey, "call-123")
	ctx = context.WithValue(ctx, SessionIDKey, "session-456")

	contextLogger := logger.WithContext(ctx)
	require.NotNil(t, contextLogger)

	// The logger should have the context values
	// This is hard to test directly, but we can verify it doesn't panic
	assert.NotPanics(t, func() {
		contextLogger.Info("test message")
	})
}

func TestLogger_SIPOperations(t *testing.T) {
	var buf bytes.Buffer

	cfg := config.LoggingConfig{
		Level:       "debug",
		Format:      "text",
		Development: true,
	}

	logger, err := NewLogger(cfg, "test")
	require.NoError(t, err)

	// Redirect output to buffer for testing
	logger.Logger = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	sipLogger := logger.SIP()
	sipLogger.Info("SIP message received", "method", "INVITE", "from", "alice@example.com")

	output := buf.String()
	assert.Contains(t, output, "SIP message received")
	assert.Contains(t, output, "method=INVITE")
	assert.Contains(t, output, "from=alice@example.com")
	assert.Contains(t, output, "operation=sip")
}

func TestLogger_RoutingOperations(t *testing.T) {
	var buf bytes.Buffer

	cfg := config.LoggingConfig{
		Level:       "debug",
		Format:      "text",
		Development: true,
	}

	logger, err := NewLogger(cfg, "test")
	require.NoError(t, err)

	logger.Logger = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	routingLogger := logger.Routing()
	routingLogger.Info("Route selected", "destination", "proxy1.example.com", "rule", "default")

	output := buf.String()
	assert.Contains(t, output, "Route selected")
	assert.Contains(t, output, "destination=proxy1.example.com")
	assert.Contains(t, output, "rule=default")
	assert.Contains(t, output, "operation=routing")
}

func TestLogger_MediaOperations(t *testing.T) {
	var buf bytes.Buffer

	cfg := config.LoggingConfig{
		Level:       "debug",
		Format:      "text",
		Development: true,
	}

	logger, err := NewLogger(cfg, "test")
	require.NoError(t, err)

	logger.Logger = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	mediaLogger := logger.Media()
	mediaLogger.Info("Media session created", "session_id", "media-123", "codec", "g711")

	output := buf.String()
	assert.Contains(t, output, "Media session created")
	assert.Contains(t, output, "session_id=media-123")
	assert.Contains(t, output, "codec=g711")
	assert.Contains(t, output, "operation=media")
}

func TestLoggerManager_Creation(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:       "info",
		Format:      "json",
		Development: false,
	}

	manager := NewLoggerManager(cfg)
	require.NotNil(t, manager)
	assert.Equal(t, cfg, manager.config)
	assert.NotNil(t, manager.loggers)
}

func TestLoggerManager_GetLogger(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:       "info",
		Format:      "text",
		Development: true,
	}

	manager := NewLoggerManager(cfg)

	// First call should create logger
	logger1, err := manager.GetLogger("component1")
	require.NoError(t, err)
	require.NotNil(t, logger1)
	assert.Equal(t, "component1", logger1.component)

	// Second call should return cached logger
	logger2, err := manager.GetLogger("component1")
	require.NoError(t, err)
	assert.Same(t, logger1, logger2)

	// Different component should get different logger
	logger3, err := manager.GetLogger("component2")
	require.NoError(t, err)
	require.NotNil(t, logger3)
	assert.Equal(t, "component2", logger3.component)
	assert.NotSame(t, logger1, logger3)
}

func TestLoggerManager_GetLogger_Error(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:  "invalid",
		Format: "text",
	}

	manager := NewLoggerManager(cfg)

	logger, err := manager.GetLogger("test")
	assert.Error(t, err)
	assert.Nil(t, logger)
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
		hasError bool
	}{
		{"debug", slog.LevelDebug, false},
		{"info", slog.LevelInfo, false},
		{"warn", slog.LevelWarn, false},
		{"error", slog.LevelError, false},
		{"DEBUG", slog.LevelDebug, false},
		{"INFO", slog.LevelInfo, false},
		{"invalid", slog.LevelInfo, true},
		{"", slog.LevelInfo, true},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			level, err := parseLogLevel(test.input)
			if test.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, level)
			}
		})
	}
}

func TestCreateHandler_Text(t *testing.T) {
	var buf bytes.Buffer

	handler := createHandler("text", &buf, slog.LevelInfo, true)
	require.NotNil(t, handler)

	logger := slog.New(handler)
	logger.Info("test message", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "key=value")
}

func TestCreateHandler_JSON(t *testing.T) {
	var buf bytes.Buffer

	handler := createHandler("json", &buf, slog.LevelInfo, false)
	require.NotNil(t, handler)

	logger := slog.New(handler)
	logger.Info("test message", "key", "value")

	output := buf.String()
	assert.Contains(t, output, `"msg":"test message"`)
	assert.Contains(t, output, `"key":"value"`)
}

func TestLogger_ContextKeys(t *testing.T) {
	// Test that context keys are properly defined
	assert.NotEqual(t, CallIDKey, SessionIDKey)
	assert.NotEqual(t, CallIDKey, RequestIDKey)
	assert.NotEqual(t, SessionIDKey, RequestIDKey)
}

func TestLogger_ConcurrentAccess(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:       "info",
		Format:      "text",
		Development: true,
	}

	logger, err := NewLogger(cfg, "concurrent-test")
	require.NoError(t, err)

	// Test concurrent logging
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				logger.Info("concurrent log", "goroutine", id, "iteration", j)
				logger.SIP().Debug("SIP log", "goroutine", id)
				logger.Routing().Warn("Routing log", "goroutine", id)
				logger.Media().Error("Media log", "goroutine", id)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// If we get here without panic, concurrent access works
}

func TestLoggerManager_ConcurrentAccess(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:       "info",
		Format:      "text",
		Development: true,
	}

	manager := NewLoggerManager(cfg)

	// Test concurrent logger creation and access
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 50; j++ {
				componentName := fmt.Sprintf("component-%d", id%3) // Reuse some component names
				logger, err := manager.GetLogger(componentName)
				assert.NoError(t, err)
				assert.NotNil(t, logger)
				logger.Info("test log", "goroutine", id, "iteration", j)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func BenchmarkLogger_Info(b *testing.B) {
	cfg := config.LoggingConfig{
		Level:       "info",
		Format:      "text",
		Development: false,
	}

	logger, _ := NewLogger(cfg, "bench")
	logger.Logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("benchmark message", "key", "value")
		}
	})
}

func BenchmarkLogger_WithContext(b *testing.B) {
	cfg := config.LoggingConfig{
		Level:       "info",
		Format:      "text",
		Development: false,
	}

	logger, _ := NewLogger(cfg, "bench")
	ctx := context.WithValue(context.Background(), CallIDKey, "call-123")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			contextLogger := logger.WithContext(ctx)
			contextLogger.Info("benchmark message")
		}
	})
}
