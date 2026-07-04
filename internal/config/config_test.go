package config

import (
	"github.com/IIAkSISII/support-assistant/internal/appdefaults"
	"strings"
	"testing"
)

func TestLoad_UsesDefaultValues(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("DEEPSEEK_API_KEY", "test-api-key")
	t.Setenv("DEEPSEEK_BASE_URL", "")
	t.Setenv("DEEPSEEK_MODEL", "")
	t.Setenv("DEEPSEEK_MAX_TOKENS", "")
	t.Setenv("KNOWLEDGE_BASE_PATH", "")
	t.Setenv("HISTORY_LIMIT", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("LOG_FORMAT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.HTTP.Addr != defaultHTTPAddr {
		t.Errorf("expected http addr %q, got %q", defaultHTTPAddr, cfg.HTTP.Addr)
	}

	if cfg.LLM.APIKey != "test-api-key" {
		t.Errorf("expected deepseek api key test-api-key, got %q", cfg.LLM.APIKey)
	}

	if cfg.LLM.BaseURL != appdefaults.DeepSeekBaseURL {
		t.Errorf("expected deepseek base url %q, got %q", appdefaults.DeepSeekBaseURL, cfg.LLM.BaseURL)
	}

	if cfg.LLM.Model != appdefaults.DeepSeekModel {
		t.Errorf("expected deepseek model %q, got %q", appdefaults.DeepSeekModel, cfg.LLM.Model)
	}

	if cfg.LLM.MaxTokens != appdefaults.DeepSeekMaxTokens {
		t.Errorf("expected deepseek max tokens %d, got %d", appdefaults.DeepSeekMaxTokens, cfg.LLM.MaxTokens)
	}

	if cfg.Knowledge.Path != defaultKnowledgeBasePath {
		t.Errorf("expected knowledge base path %q, got %q", defaultKnowledgeBasePath, cfg.Knowledge.Path)
	}

	if cfg.History.Limit != appdefaults.HistoryLimit {
		t.Errorf("expected history limit %d, got %d", appdefaults.HistoryLimit, cfg.History.Limit)
	}
	
	if cfg.Logger.Level != defaultLogLevel {
		t.Errorf("expected log level %q, got %q", defaultLogLevel, cfg.Logger.Level)
	}

	if cfg.Logger.Format != defaultLogFormat {
		t.Errorf("expected log format %q, got %q", defaultLogFormat, cfg.Logger.Format)
	}
}

func TestLoad_UsesEnvValues(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":9000")
	t.Setenv("DEEPSEEK_API_KEY", "real-api-key")
	t.Setenv("DEEPSEEK_BASE_URL", "https://custom.deepseek.example.com")
	t.Setenv("DEEPSEEK_MODEL", "deepseek-reasoner")
	t.Setenv("DEEPSEEK_MAX_TOKENS", "2048")
	t.Setenv("KNOWLEDGE_BASE_PATH", "./data/knowledge_base.json")
	t.Setenv("HISTORY_LIMIT", "25")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.HTTP.Addr != ":9000" {
		t.Errorf("expected http addr :9000, got %q", cfg.HTTP.Addr)
	}

	if cfg.LLM.APIKey != "real-api-key" {
		t.Errorf("expected deepseek api key real-api-key, got %q", cfg.LLM.APIKey)
	}

	if cfg.LLM.BaseURL != "https://custom.deepseek.example.com" {
		t.Errorf("unexpected deepseek base url: %q", cfg.LLM.BaseURL)
	}

	if cfg.LLM.Model != "deepseek-reasoner" {
		t.Errorf("expected deepseek model deepseek-reasoner, got %q", cfg.LLM.Model)
	}

	if cfg.LLM.MaxTokens != 2048 {
		t.Errorf("expected deepseek max tokens 2048, got %d", cfg.LLM.MaxTokens)
	}

	if cfg.Knowledge.Path != "./data/knowledge_base.json" {
		t.Errorf("unexpected knowledge base path: %q", cfg.Knowledge.Path)
	}

	if cfg.History.Limit != 25 {
		t.Errorf("expected history limit 25, got %d", cfg.History.Limit)
	}
}

func TestLoad_ReturnsErrorWhenDeepSeekAPIKeyIsMissing(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("DEEPSEEK_API_KEY", "")
	t.Setenv("DEEPSEEK_BASE_URL", "")
	t.Setenv("DEEPSEEK_MODEL", "")
	t.Setenv("DEEPSEEK_MAX_TOKENS", "")
	t.Setenv("KNOWLEDGE_BASE_PATH", "")
	t.Setenv("HISTORY_LIMIT", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "DEEPSEEK_API_KEY is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoad_ReturnsErrorWhenDeepSeekMaxTokensIsInvalid(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "test-api-key")
	t.Setenv("DEEPSEEK_MAX_TOKENS", "invalid")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "DEEPSEEK_MAX_TOKENS must be integer") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoad_ReturnsErrorWhenDeepSeekMaxTokensIsNotPositive(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "test-api-key")
	t.Setenv("DEEPSEEK_MAX_TOKENS", "0")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "DEEPSEEK_MAX_TOKENS must be positive") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoad_ReturnsErrorWhenHistoryLimitIsInvalid(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "test-api-key")
	t.Setenv("DEEPSEEK_MAX_TOKENS", "")
	t.Setenv("HISTORY_LIMIT", "invalid")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "HISTORY_LIMIT must be integer") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoad_ReturnsErrorWhenHistoryLimitIsNotPositive(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "test-api-key")
	t.Setenv("DEEPSEEK_MAX_TOKENS", "")
	t.Setenv("HISTORY_LIMIT", "-1")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "HISTORY_LIMIT must be positive") {
		t.Errorf("unexpected error: %v", err)
	}
}
