package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/IIAkSISII/support-assistant/internal/model"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultHTTPClientTimeout = 10 * time.Second

type Config struct {
	BaseURL        string
	APIAccessToken string
	HTTPClient     *http.Client
}

type Sender interface {
	SendProcessResult(
		ctx context.Context,
		accountID int64,
		conversationID int64,
		result model.ProcessResult) error
}

type MessageSender struct {
	baseURL        string
	apiAccessToken string
	httpClient     *http.Client
}

func NewMessageSender(config Config) (*MessageSender, error) {
	baseUrl := strings.TrimSpace(config.BaseURL)
	apiAccessToken := strings.TrimSpace(config.APIAccessToken)
	httpClient := config.HTTPClient

	if baseUrl == "" {
		return nil, errors.New("chatwoot base url is required")
	}

	if apiAccessToken == "" {
		return nil, errors.New("chatwoot api access token is required")
	}

	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: defaultHTTPClientTimeout,
		}
	}

	return &MessageSender{
		baseURL:        strings.TrimRight(baseUrl, "/"),
		apiAccessToken: apiAccessToken,
		httpClient:     httpClient,
	}, nil
}

func (s *MessageSender) SendProcessResult(
	ctx context.Context,
	accountID int64,
	conversationID int64,
	result model.ProcessResult,
) error {
	if accountID <= 0 {
		return errors.New("account id is required")
	}
	if conversationID <= 0 {
		return errors.New("conversation id is required")
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal process result: %w", err)
	}

	payload := messageRequest{
		Content:           string(resultJSON),
		MessageType:       "outgoing",
		Private:           false,
		ContentType:       "text",
		ContentAttributes: map[string]any{},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal chatwoot message request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/accounts/%d/conversations/%d/messages",
		s.baseURL,
		accountID,
		conversationID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create chatwoot request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api_access_token", s.apiAccessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send chatwoot request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read chatwoot response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("chatwoot response code %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

type messageRequest struct {
	Content           string         `json:"content"`
	MessageType       string         `json:"message_type"`
	Private           bool           `json:"private"`
	ContentType       string         `json:"content_type"`
	ContentAttributes map[string]any `json:"content_attributes"`
}
