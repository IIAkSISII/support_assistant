package history

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/IIAkSISII/support-assistant/internal/model"
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
)

const (
	defaultRedisKeyPrefix       = "support_assistant:history"
	defaultOperationTimeout     = 3 * time.Second
	defaultMaxMessagesPerDialog = 10
)

type RedisConfig struct {
	KeyPrefix        string
	MaxMessages      int
	OperationTimeout time.Duration
}

type redisRepository struct {
	client           *redis.Client
	keyPrefix        string
	maxMessages      int
	operationTimeout time.Duration
}

func NewHistoryRedisRepository(client *redis.Client, config RedisConfig) (HistoryRepository, error) {
	if client == nil {
		return nil, errors.New("redis client is nil")
	}

	keyPrefix := strings.TrimSpace(config.KeyPrefix)
	if keyPrefix == "" {
		keyPrefix = defaultRedisKeyPrefix
	}

	maxMessages := config.MaxMessages
	if maxMessages <= 0 {
		maxMessages = defaultMaxMessagesPerDialog
	}

	operationTimeout := config.OperationTimeout
	if operationTimeout <= 0 {
		operationTimeout = defaultOperationTimeout
	}

	repository := &redisRepository{
		client:           client,
		keyPrefix:        keyPrefix,
		maxMessages:      maxMessages,
		operationTimeout: operationTimeout,
	}

	ctx, cancel := repository.context()
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return repository, nil
}

func (r *redisRepository) AddMessage(dialogID string, message model.Message) error {
	dialogID = strings.TrimSpace(dialogID)
	if dialogID == "" {
		return errors.New("dialogID is required")
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal history message: %w", err)
	}

	ctx, cancel := r.context()
	defer cancel()

	key := r.key(dialogID)

	pipe := r.client.TxPipeline()
	pipe.RPush(ctx, key, payload)
	pipe.LTrim(ctx, key, -int64(r.maxMessages), -1)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("save message to redis history: %w", err)
	}

	return nil
}

func (r *redisRepository) GetLastMessages(dialogID string, limit int) ([]model.Message, error) {
	dialogID = strings.TrimSpace(dialogID)
	if dialogID == "" {
		return []model.Message{}, nil
	}

	if limit <= 0 {
		return []model.Message{}, nil
	}

	ctx, cancel := r.context()
	defer cancel()

	rawMessages, err := r.client.LRange(ctx, r.key(dialogID), -int64(limit), -1).Result()
	if err != nil {
		return nil, fmt.Errorf("read messages from redis history: %w", err)
	}

	messages := make([]model.Message, 0, len(rawMessages))
	for _, rawMessage := range rawMessages {
		var message model.Message
		if err := json.Unmarshal([]byte(rawMessage), &message); err != nil {
			return nil, fmt.Errorf("unmarshal history message from redis: %w", err)
		}

		messages = append(messages, message)
	}

	return messages, nil
}

func (r *redisRepository) context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), r.operationTimeout)
}

func (r *redisRepository) key(dialogID string) string {
	return r.keyPrefix + ":" + dialogID
}
