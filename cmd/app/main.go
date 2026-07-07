package main

import (
	"context"
	"errors"
	_ "github.com/IIAkSISII/support-assistant/docs"
	"github.com/IIAkSISII/support-assistant/internal/client"
	"github.com/IIAkSISII/support-assistant/internal/config"
	"github.com/IIAkSISII/support-assistant/internal/handler"
	"github.com/IIAkSISII/support-assistant/internal/llm"
	appLogger "github.com/IIAkSISII/support-assistant/internal/logger"
	"github.com/IIAkSISII/support-assistant/internal/middleware"
	"github.com/IIAkSISII/support-assistant/internal/repository/history"
	"github.com/IIAkSISII/support-assistant/internal/repository/knowledge"
	"github.com/IIAkSISII/support-assistant/internal/service"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 10 * time.Second
)

// @title           Ассистент поддержки
// @version         1.0
// @description     Webhook для классификации запросов в поддержку, выбора готовых ответов и подготовки контекста эскалации для операторов.
// @BasePath        /
// @schemes         http
// @accept			json
func main() {
	if err := run(); err != nil {
		slog.Error("application failed", "error", err.Error())
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger, err := appLogger.NewLogger(os.Stdout, cfg.Logger.Level, cfg.Logger.Format)
	if err != nil {
		return err
	}

	slog.SetDefault(logger)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Warn("redis close failed", "error", err.Error())
		}
	}()

	historyRepository, err := history.NewHistoryRedisRepository(redisClient, history.RedisConfig{
		KeyPrefix:        cfg.Redis.KeyPrefix,
		MaxMessages:      cfg.History.Limit,
		OperationTimeout: time.Duration(cfg.Redis.OperationTimeoutSeconds) * time.Second,
	})
	if err != nil {
		return err
	}

	knowledgeRepository, err := knowledge.NewJsonRepository(cfg.Knowledge.Path)
	if err != nil {
		return err
	}

	analyzer, err := llm.NewAnalyzer(llm.Config{
		APIKey:    cfg.LLM.APIKey,
		BaseURL:   cfg.LLM.BaseURL,
		Model:     cfg.LLM.Model,
		MaxTokens: cfg.LLM.MaxTokens,
		HTTPClient: &http.Client{
			Timeout: time.Duration(cfg.LLM.TimeoutSeconds) * time.Second,
		},
	})
	if err != nil {
		return err
	}

	processor := service.NewMessageProcessor(
		historyRepository,
		knowledgeRepository,
		analyzer,
		cfg.History.Limit,
	)

	var resultSender client.Sender

	if cfg.Chatwoot.Enabled {
		resultSender, err = client.NewMessageSender(client.Config{
			BaseURL:        cfg.Chatwoot.BaseURL,
			APIAccessToken: cfg.Chatwoot.APIAccessToken,
		})
		if err != nil {
			return err
		}
	}

	webhookHandler := handler.NewWebhookHandlerWithSender(processor, resultSender, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", webhookHandler.HandleWebhook)
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	httpHandler := middleware.RequestLogger(logger)(mux)

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           httpHandler,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	serverErrors := make(chan error, 1)

	go func() {
		logger.Info("http server started", "addr", cfg.HTTP.Addr)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}

		close(serverErrors)
	}()

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-shutdownCtx.Done():
		logger.Info("shutdown signal received")
	case err := <-serverErrors:
		if err != nil {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return err
	}

	logger.Info("http server stopped")

	return nil
}
