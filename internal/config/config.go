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
	defaultHTTPAddr              = ":8090"
	defaultKnowledgeBasePath     = "internal/repository/knowledge/testdata/knowledge_base.json"
	defaultLogLevel              = "info"
	defaultLogFormat             = "json"
	defaultRedisAddr             = "redis:6379"
	defaultRedisKeyPrefix        = "support_assistant:history"
	defaultRedisOperationTimeout = 3
)

type Config struct {
	HTTP            HTTPConfig
	LLM             LLMConfig
	Knowledge       KnowledgeConfig
	History         HistoryConfig
	Logger          LoggerConfig
	SupportPlatform SupportPlatformConfig
	Redis           RedisConfig
}

type HTTPConfig struct {
	Addr string
}

type LLMConfig struct {
	APIKey           string
	BaseURL          string
	Model            string
	MaxTokens        int
	TimeoutSeconds   int
	SystemPromptPath string
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

type SupportPlatformConfig struct {
	Enabled        bool
	BaseURL        string
	APIAccessToken string
}

type RedisConfig struct {
	Addr                    string
	Password                string
	DB                      int
	KeyPrefix               string
	OperationTimeoutSeconds int
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

	supportPlatformEnabled, err := getEnvBool("SUPPORT_PLATFORM_ENABLED", false)
	if err != nil {
		return Config{}, err
	}

	redisDB, err := getEnvNonNegativeInt("REDIS_DB", 0)
	if err != nil {
		return Config{}, err
	}

	redisOperationTimeoutSeconds, err := getEnvInt("REDIS_OPERATION_TIMEOUT_SECONDS", defaultRedisOperationTimeout)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		HTTP: HTTPConfig{
			Addr: getEnv("HTTP_ADDR", defaultHTTPAddr),
		},
		LLM: LLMConfig{
			APIKey:           getEnv("LLM_API_KEY", ""),
			BaseURL:          getEnv("LLM_BASE_URL", appdefaults.LLMbaseURL),
			Model:            getEnv("LLM_MODEL", appdefaults.LLMmodel),
			MaxTokens:        deepSeekMaxTokens,
			TimeoutSeconds:   llmTimeoutSeconds,
			SystemPromptPath: getEnv("SYSTEM_PROMPT_PATH", "prompts/system.md"),
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
		SupportPlatform: SupportPlatformConfig{
			Enabled:        supportPlatformEnabled,
			BaseURL:        getEnv("SUPPORT_PLATFORM_BASE_URL", "https://guiai-test.ru"),
			APIAccessToken: getEnv("SUPPORT_PLATFORM_API_ACCESS_TOKEN", ""),
		},
		Redis: RedisConfig{
			Addr:                    getEnv("REDIS_ADDR", defaultRedisAddr),
			Password:                getEnv("REDIS_PASSWORD", ""),
			DB:                      redisDB,
			KeyPrefix:               getEnv("REDIS_KEY_PREFIX", defaultRedisKeyPrefix),
			OperationTimeoutSeconds: redisOperationTimeoutSeconds,
		},
	}

	if config.LLM.APIKey == "" {
		return Config{}, errors.New("LLM_API_KEY is required")
	}
	if config.SupportPlatform.Enabled && config.SupportPlatform.APIAccessToken == "" {
		return Config{}, errors.New("SUPPORT_PLATFORM_API_ACCESS_TOKEN is required when SUPPORT_PLATFORM_ENABLED=true")
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

func getEnvNonNegativeInt(key string, defaultValue int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be integer: %w", key, err)
	}

	if parsed < 0 {
		return 0, fmt.Errorf("%s must be non-negative", key)
	}

	return parsed, nil
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
