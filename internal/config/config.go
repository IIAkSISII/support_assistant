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
	defaultHTTPAddr          = ":8080"
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
}

type HTTPConfig struct {
	Addr string
}

type LLMConfig struct {
	APIKey    string
	BaseURL   string
	Model     string
	MaxTokens int
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

func Load() (Config, error) {
	deepSeekMaxTokens, err := getEnvInt("DEEPSEEK_MAX_TOKENS", appdefaults.DeepSeekMaxTokens)
	if err != nil {
		return Config{}, err
	}

	historyLimit, err := getEnvInt("HISTORY_LIMIT", appdefaults.HistoryLimit)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		HTTP: HTTPConfig{
			Addr: getEnv("HTTP_ADDR", defaultHTTPAddr),
		},
		LLM: LLMConfig{
			APIKey:    getEnv("DEEPSEEK_API_KEY", ""),
			BaseURL:   getEnv("DEEPSEEK_BASE_URL", appdefaults.DeepSeekBaseURL),
			Model:     getEnv("DEEPSEEK_MODEL", appdefaults.DeepSeekModel),
			MaxTokens: deepSeekMaxTokens,
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
	}

	if config.LLM.APIKey == "" {
		return Config{}, errors.New("DEEPSEEK_API_KEY is required")
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
