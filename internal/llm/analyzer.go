package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/IIAkSISII/support-assistant/internal/appdefaults"
	"github.com/IIAkSISII/support-assistant/internal/model"
	"io"
	"net/http"
	"strings"
	"time"
)

type Analyzer interface {
	Analyze(ctx context.Context, request model.AnalysisRequest) (model.AnalysisResult, error)
}

type Config struct {
	APIKey     string
	BaseURL    string
	Model      string
	MaxTokens  int
	HTTPClient *http.Client
}

type LLMAnalyzer struct {
	apiKey     string
	baseURL    string
	model      string
	maxTokens  int
	httpClient *http.Client
}

func NewAnalyzer(config Config) (*LLMAnalyzer, error) {
	if config.APIKey == "" {
		return nil, errors.New("deepseek api key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = appdefaults.LLMbaseURL
	}

	if config.Model == "" {
		config.Model = appdefaults.LLMmodel
	}

	if config.MaxTokens <= 0 {
		config.MaxTokens = appdefaults.LLMmaxTokens
	}

	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{
			Timeout: 60 * time.Second,
		}
	}

	return &LLMAnalyzer{
		apiKey:     config.APIKey,
		baseURL:    strings.TrimRight(config.BaseURL, "/"),
		model:      config.Model,
		maxTokens:  config.MaxTokens,
		httpClient: config.HTTPClient,
	}, nil
}

func (a *LLMAnalyzer) Analyze(ctx context.Context, request model.AnalysisRequest) (model.AnalysisResult, error) {
	messages, err := buildMessages(request)
	if err != nil {
		return model.AnalysisResult{}, err
	}

	body := chatCompletionRequest{
		Model:    a.model,
		Messages: messages,
		ResponseFormat: responseFormat{
			Type: "json_object",
		},
		MaxTokens: a.maxTokens,
		Stream:    false,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return model.AnalysisResult{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return model.AnalysisResult{}, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	httpResp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return model.AnalysisResult{}, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return model.AnalysisResult{}, fmt.Errorf("read response body: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return model.AnalysisResult{}, fmt.Errorf("got response code %d: %s", httpResp.StatusCode, string(responseBody))
	}

	var chatResponse chatCompletionResponse

	if err := json.Unmarshal(responseBody, &chatResponse); err != nil {
		return model.AnalysisResult{}, fmt.Errorf("unmarshal response: %w", err)
	}
	if len(chatResponse.Choices) == 0 {
		return model.AnalysisResult{}, errors.New("response has no choices")
	}

	choice := chatResponse.Choices[0]

	if choice.FinishReason != "" && choice.FinishReason != "stop" {
		return model.AnalysisResult{}, fmt.Errorf("deepseek finished with reason %q", choice.FinishReason)
	}

	content := strings.TrimSpace(choice.Message.Content)
	if content == "" {
		return model.AnalysisResult{}, errors.New("response content is empty")
	}

	analysis, err := parseAnalysisResult(content)
	if err != nil {
		return model.AnalysisResult{}, fmt.Errorf("parse analysis result: %w", err)
	}

	return analysis, nil
}

func buildMessages(request model.AnalysisRequest) ([]chatMessage, error) {
	userPrompt, err := json.MarshalIndent(request, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal analysis request: %w", err)
	}
	return []chatMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: string(userPrompt),
		},
	}, nil
}

func parseAnalysisResult(content string) (model.AnalysisResult, error) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var result model.AnalysisResult

	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return model.AnalysisResult{}, fmt.Errorf("unmarshal analysis result: %w", err)
	}
	if result.Category == "" {
		result.Category = "unknown"
	}
	switch result.Priority {
	case "low", "medium", "high", "critical":
	default:
		result.Priority = "medium"
	}
	if result.Keywords == nil {
		result.Keywords = []string{}
	}

	return result, nil
}

type chatCompletionRequest struct {
	Model          string         `json:"model"`
	Messages       []chatMessage  `json:"messages"`
	ResponseFormat responseFormat `json:"response_format"`
	MaxTokens      int            `json:"max_tokens"`
	Stream         bool           `json:"stream"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatCompletionResponse struct {
	Choices []chatChoice `json:"choices"`
}

type chatChoice struct {
	FinishReason string      `json:"finish_reason"`
	Message      chatMessage `json:"message"`
}

const systemPrompt = `
Ты классификатор и маршрутизатор обращений в службу поддержки.

Верни только один валидный JSON-объект.
Не пиши markdown, пояснения или текст вне JSON.
Не добавляй поля вне схемы.

Ты НЕ отвечаешь пользователю.
Ты НЕ придумываешь факты о системе.
Ты только анализируешь обращение и решаешь, нужен ли оператор.

Используй knowledge_entries для выбора category и keywords.
Наличие записи в knowledge_entries НЕ означает, что обращение можно закрыть без оператора.

category:
- выбирай только из knowledge_entries.category;
- если подходящей категории нет, верни "unknown";

keywords:
- всегда массив строк;
- Keywords по возможности выбирай из keywords подходящей записи knowledge_entries. Можно выбрать один или несколько keywords, если они действительно соответствуют сообщению пользователя.
- если нет подходящих, верни [].

escalate:
- true, если требуется проверка сотрудником;
- true, если обращение связано с проблемой оплаты, платежа, подписки, списания, возврата или доступа после оплаты;
- true, если пользователь сообщает, что действие не сработало: оплата не прошла, доступ не появился, пароль не восстанавливается, ошибка повторяется;
- true, если пользователь прислал email, номер платежа, чек, transaction id, номер заказа, скриншот или другие данные для проверки;
- true, если нужно проверить аккаунт, платеж, доступ, личные данные, админ-панель или внутреннее состояние системы;
- true, если нет точного готового ответа или обращение неоднозначное;
- false только для простого типового информационного вопроса, который можно безопасно закрыть готовым ответом;
- если сомневаешься, ставь true.

priority:
- low: простой информационный вопрос;
- medium: обычная проблема без полной блокировки;
- high: проблема с оплатой, доступом, аккаунтом, подпиской или требуется проверка сотрудником;
- critical: массовый сбой, недоступность сервиса, безопасность, потеря данных или массовый финансовый инцидент.

summary:
- кратко опиши суть обращения.

reason:
- reason должно объяснять, почему нужна или не нужна передача оператору.

suggest_action:
- если escalate = true, напиши конкретное действие для оператора. Действие должно объяснять, что именно проверить, запросить или сделать дальше.
- если escalate = false, верни "".

Верни json строго такого вида:
{
  "category": "<category или unknown>",
  "priority": "<low|medium|high|critical>",
  "keywords": ["<keyword>"],
  "escalate": false,
  "summary": "<краткое описание>",
  "reason": "<причина решения>",
  "suggest_action": "<действие для оператора или пустая строка>"
}
`
