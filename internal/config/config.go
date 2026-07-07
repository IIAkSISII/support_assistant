package config

import (
	"errors"
	"fmt"
	"github.com/IIAkSISII/support-assistant/internal/appdefaults"
	"os"
	"strconv"
	"strings"
)

const (
	defaultHTTPAddr          = ":8090"
	defaultKnowledgeBasePath = "internal/repository/knowledge/testdata/knowledge_base.json"
	defaultLogLevel          = "info"
	defaultLogFormat         = "json"
)

type Config struct {
	HTTP      HTTPConfig
	LLM       LLMConfig
	Knowledge KnowledgeConfig
	History   HistoryConfig
	Logger    LoggerConfig
	Chatwoot  ChatwootConfig
}

type HTTPConfig struct {
	Addr string
}

type LLMConfig struct {
	APIKey         string
	BaseURL        string
	Model          string
	MaxTokens      int
	TimeoutSeconds int
}

type KnowledgeConfig struct {
	Path string
}

type HistoryConfig struct {
	Limit int
}

type LoggerConfig struct {
	Level  string
	Format string
}

type ChatwootConfig struct {
	Enabled        bool
	BaseURL        string
	APIAccessToken string
}

func Load() (Config, error) {
	deepSeekMaxTokens, err := getEnvInt("LLM_MAX_TOKENS", appdefaults.LLMmaxTokens)
	if err != nil {
		return Config{}, err
	}

	historyLimit, err := getEnvInt("HISTORY_LIMIT", appdefaults.HistoryLimit)
	if err != nil {
		return Config{}, err
	}

	llmTimeoutSeconds, err := getEnvInt("LLM_TIMEOUT_SECONDS", 60)
	if err != nil {
		return Config{}, err
	}

	chatwootEnabled, err := getEnvBool("CHATWOOT_ENABLED", false)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		HTTP: HTTPConfig{
			Addr: getEnv("HTTP_ADDR", defaultHTTPAddr),
		},
		LLM: LLMConfig{
			APIKey:         getEnv("LLM_API_KEY", ""),
			BaseURL:        getEnv("LLM_BASE_URL", appdefaults.LLMbaseURL),
			Model:          getEnv("LLM_MODEL", appdefaults.LLMmodel),
			MaxTokens:      deepSeekMaxTokens,
			TimeoutSeconds: llmTimeoutSeconds,
		},
		Knowledge: KnowledgeConfig{
			Path: getEnv("KNOWLEDGE_BASE_PATH", defaultKnowledgeBasePath),
		},
		History: HistoryConfig{
			Limit: historyLimit,
		},
		Logger: LoggerConfig{
			Level:  getEnv("LOG_LEVEL", defaultLogLevel),
			Format: getEnv("LOG_FORMAT", defaultLogFormat),
		},
		Chatwoot: ChatwootConfig{
			Enabled:        chatwootEnabled,
			BaseURL:        getEnv("CHATWOOT_BASE_URL", "https://guiai-test.ru"),
			APIAccessToken: getEnv("CHATWOOT_API_ACCESS_TOKEN", ""),
		},
	}

	if config.LLM.APIKey == "" {
		return Config{}, errors.New("LLM_API_KEY is required")
	}
	if config.Chatwoot.Enabled && config.Chatwoot.APIAccessToken == "" {
		return Config{}, errors.New("CHATWOOT_API_ACCESS_TOKEN is required when ENABLED=true")
	}

	return config, nil
}

func getEnv(key string, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}

	return value
}

func getEnvInt(key string, defaultValue int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be integer: %w", key, err)
	}

	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}

	return parsed, nil
}

func getEnvBool(key string, defaultValue bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be boolean: %w", key, err)
	}

	return parsed, nil
}
