package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewLogger_CreatesJSONLogger(t *testing.T) {
	var output bytes.Buffer

	log, err := NewLogger(&output, "info", "json")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	log.Info("service started", "addr", ":8080")

	result := output.String()

	if !strings.Contains(result, `"msg":"service started"`) {
		t.Errorf("expected log message in output, got %s", result)
	}

	if !strings.Contains(result, `"addr":":8080"`) {
		t.Errorf("expected addr field in output, got %s", result)
	}
}

func TestNewLogger_CreatesTextLogger(t *testing.T) {
	var output bytes.Buffer

	log, err := NewLogger(&output, "info", "text")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	log.Info("service started", "addr", ":8080")

	result := output.String()

	if !strings.Contains(result, "service started") {
		t.Errorf("expected log message in output, got %s", result)
	}

	if !strings.Contains(result, "addr=:8080") {
		t.Errorf("expected addr field in output, got %s", result)
	}
}

func TestNewLogger_FiltersDebugMessagesWhenLevelIsInfo(t *testing.T) {
	var output bytes.Buffer

	log, err := NewLogger(&output, "info", "json")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	log.Debug("hidden debug message")

	if strings.Contains(output.String(), "hidden debug message") {
		t.Errorf("expected debug message to be filtered, got %s", output.String())
	}
}

func TestNewLogger_AllowsDebugMessagesWhenLevelIsDebug(t *testing.T) {
	var output bytes.Buffer

	log, err := NewLogger(&output, "debug", "json")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	log.Debug("visible debug message")

	if !strings.Contains(output.String(), "visible debug message") {
		t.Errorf("expected debug message in output, got %s", output.String())
	}
}

func TestNewLogger_ReturnsErrorForUnsupportedLevel(t *testing.T) {
	var output bytes.Buffer

	_, err := NewLogger(&output, "trace", "json")
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "unsupported log level") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewLogger_ReturnsErrorForUnsupportedFormat(t *testing.T) {
	var output bytes.Buffer

	_, err := NewLogger(&output, "info", "xml")
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "unsupported log format") {
		t.Errorf("unexpected error: %v", err)
	}
}
