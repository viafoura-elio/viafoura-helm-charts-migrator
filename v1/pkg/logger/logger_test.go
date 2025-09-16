package logger

import (
	"flag"
	"testing"
)

func TestInitFlags(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	InitFlags(fs)

	// Check that klog flags are registered
	if fs.Lookup("v") == nil {
		t.Error("expected verbosity flag 'v' to be registered")
	}

	if fs.Lookup("logtostderr") == nil {
		t.Error("expected 'logtostderr' flag to be registered")
	}
}

func TestNamedLogger(t *testing.T) {
	tests := []struct {
		name     string
		logName  string
		validate func(*NamedLogger)
	}{
		{
			name:    "create named logger",
			logName: "test-component",
			validate: func(l *NamedLogger) {
				if l.name != "test-component" {
					t.Errorf("expected name 'test-component', got %s", l.name)
				}
			},
		},
		{
			name:    "empty name",
			logName: "",
			validate: func(l *NamedLogger) {
				if l.name != "" {
					t.Errorf("expected empty name, got %s", l.name)
				}
			},
		},
		{
			name:    "special characters in name",
			logName: "test/component:v1.0",
			validate: func(l *NamedLogger) {
				if l.name != "test/component:v1.0" {
					t.Errorf("expected name 'test/component:v1.0', got %s", l.name)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := WithName(tt.logName)
			if logger == nil {
				t.Fatal("expected non-nil logger")
			}
			tt.validate(logger)
		})
	}
}

func TestNamedLogger_NoPanic(t *testing.T) {
	// Test that logging functions don't panic
	logger := WithName("test-logger")

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("logger function panicked: %v", r)
		}
	}()

	// Test all logging methods
	logger.Info("test info message")
	logger.InfoS("structured log", "key1", "value1", "key2", 123)
	logger.Warning("warning message")
	logger.Error(&testError{msg: "test error"}, "error message", "key", "value")

	// Test with empty messages
	logger.Info("")
	logger.InfoS("", "key", "value")
	logger.Warning("")

	// Test verbosity
	v := logger.V(2)
	_ = v // Just ensure it doesn't panic
}

func TestGlobalLogging_NoPanic(t *testing.T) {
	// Test that global logging functions don't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("global logging function panicked: %v", r)
		}
	}()

	// Test all global functions
	Info("global info message", "key", "value")
	InfoS("structured info", "count", 42)
	Warning("global warning")
	WarningS("structured warning", "level", "high")
	Error(&testError{msg: "test"}, "error occurred", "operation", "test-op")
	ErrorS(&testError{msg: "test"}, "structured error", "code", 500)

	// Test with empty messages
	Info("")
	InfoS("")
	Warning("")
	WarningS("")

	// Test verbosity
	v := V(2)
	_ = v // Just ensure it doesn't panic
}

func TestVerbosity(t *testing.T) {
	// Test V() function returns proper verbose logger
	v := V(2)
	// klog.Verbose is a struct type, not a pointer, so we can't check for nil
	// Just verify it doesn't panic
	_ = v

	// Test NamedLogger V() function
	logger := WithName("test")
	v = logger.V(3)
	// klog.Verbose is a struct type, not a pointer, so we can't check for nil
	// Just verify it doesn't panic
	_ = v
}

func TestFlush(t *testing.T) {
	// Test that Flush doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Flush() panicked: %v", r)
		}
	}()

	Flush()
}

func TestMultipleNamedLoggers(t *testing.T) {
	// Test that multiple named loggers can be created without issues
	loggers := []*NamedLogger{
		WithName("auth"),
		WithName("database"),
		WithName("api"),
		WithName("cache"),
	}

	for i, logger := range loggers {
		if logger == nil {
			t.Errorf("logger %d is nil", i)
		}
	}

	// Test that they can all log without panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("logging panicked: %v", r)
		}
	}()

	for _, logger := range loggers {
		logger.Info("test message")
	}
}

func TestNamedLoggerChaining(t *testing.T) {
	// Test that named loggers can be created and used in sequence
	loggers := []*NamedLogger{
		WithName("auth"),
		WithName("database"),
		WithName("api"),
		WithName("cache"),
	}

	for i, logger := range loggers {
		if logger == nil {
			t.Errorf("logger %d is nil", i)
		}
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func BenchmarkNamedLogger(b *testing.B) {
	logger := WithName("benchmark")

	b.Run("Info", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Info("benchmark message")
		}
	})

	b.Run("InfoS", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.InfoS("benchmark message", "key1", "value1", "key2", 42)
		}
	})

	b.Run("Warning", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Warning("benchmark warning")
		}
	})
}

func BenchmarkGlobalLogger(b *testing.B) {
	b.Run("Info", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Info("benchmark message")
		}
	})

	b.Run("InfoS", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			InfoS("benchmark message", "key1", "value1", "key2", 42)
		}
	})

	b.Run("Warning", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Warning("benchmark warning")
		}
	})
}
