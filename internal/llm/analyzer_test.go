package llm

import (
	"context"
	"encoding/json"
	"github.com/IIAkSISII/support-assistant/internal/model"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestNewAnalyzer_RequiresAPIKey(t *testing.T) {
	analyzer, err := NewAnalyzer(Config{})
	if err == nil {
		t.Fatal("expected error for missing api key")
	}

	if analyzer != nil {
		t.Fatal("expected analyzer to be nil")
	}

	if err.Error() != "deepseek api key is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAnalyze_UsesDefaultConfigurationAndParsesSuccessfulResponse(t *testing.T) {
	request := model.AnalysisRequest{
		DialogID: "dlg_1",
		UserID:   "user_1",
		Message:  "Не могу войти в аккаунт",
		History: []model.Message{
			{Role: "user", Content: "Не могу войти"},
		},
		KnowledgeEntries: []model.Entry{
			{Category: "auth", Keywords: []string{"логин", "пароль"}, Answer: "Попробуйте восстановить пароль."},
		},
	}

	var gotMethod string
	var gotPath string
	var gotAuthHeader string
	var gotContentType string
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuthHeader = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}

		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"finish_reason": "stop",
					"message": {
						"role": "assistant",
						"content": "{\"category\":\"auth\",\"priority\":\"high\",\"keywords\":[\"логин\",\"пароль\"],\"escalate\":false,\"summary\":\"Проблема со входом.\",\"reason\":\"Есть типовой сценарий.\",\"suggest_action\":\"\"}"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	analyzer, err := NewAnalyzer(Config{
		APIKey:  "secret-token",
		BaseURL: server.URL + "/",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	result, err := analyzer.Analyze(context.Background(), request)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method POST, got %s", gotMethod)
	}

	if gotPath != "/chat/completions" {
		t.Fatalf("expected path /chat/completions, got %s", gotPath)
	}

	if gotAuthHeader != "Bearer secret-token" {
		t.Fatalf("expected auth header %q, got %q", "Bearer secret-token", gotAuthHeader)
	}

	if gotContentType != "application/json" {
		t.Fatalf("expected content type %q, got %q", "application/json", gotContentType)
	}

	if gotBody["model"] != "deepseek-v4-flash" {
		t.Fatalf("expected default model, got %#v", gotBody["model"])
	}

	if gotBody["max_tokens"] != float64(2000) {
		t.Fatalf("expected default max_tokens 2000, got %#v", gotBody["max_tokens"])
	}

	if gotBody["stream"] != false {
		t.Fatalf("expected stream false, got %#v", gotBody["stream"])
	}

	responseFormat, ok := gotBody["response_format"].(map[string]any)
	if !ok {
		t.Fatalf("expected response_format object, got %#v", gotBody["response_format"])
	}

	if responseFormat["type"] != "json_object" {
		t.Fatalf("expected response_format.type json_object, got %#v", responseFormat["type"])
	}

	messages, ok := gotBody["messages"].([]any)
	if !ok {
		t.Fatalf("expected messages array, got %#v", gotBody["messages"])
	}

	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	systemMessage, ok := messages[0].(map[string]any)
	if !ok {
		t.Fatalf("expected system message object, got %#v", messages[0])
	}

	if systemMessage["role"] != "system" {
		t.Fatalf("expected first role system, got %#v", systemMessage["role"])
	}

	userMessage, ok := messages[1].(map[string]any)
	if !ok {
		t.Fatalf("expected user message object, got %#v", messages[1])
	}

	if userMessage["role"] != "user" {
		t.Fatalf("expected second role user, got %#v", userMessage["role"])
	}

	userContent, ok := userMessage["content"].(string)
	if !ok {
		t.Fatalf("expected user content string, got %#v", userMessage["content"])
	}

	if !strings.Contains(userContent, `"dialog_id": "dlg_1"`) {
		t.Fatalf("expected request payload to include dialog id, got %q", userContent)
	}

	if !strings.Contains(userContent, `"message": "Не могу войти в аккаунт"`) {
		t.Fatalf("expected request payload to include message, got %q", userContent)
	}

	if result.Category != "auth" {
		t.Fatalf("expected category auth, got %q", result.Category)
	}

	if result.Priority != "high" {
		t.Fatalf("expected priority high, got %q", result.Priority)
	}

	if !reflect.DeepEqual(result.Keywords, []string{"логин", "пароль"}) {
		t.Fatalf("unexpected keywords: %#v", result.Keywords)
	}

	if result.Escalate {
		t.Fatal("expected escalate to be false")
	}
}

func TestParseAnalysisResult_NormalizesModelOutput(t *testing.T) {
	content := "```json\n" +
		`{"category":"","priority":"urgent","escalate":true,"summary":"Нужна помощь","reason":"Сложный случай","suggest_action":"Проверить вручную"}` +
		"\n```"

	result, err := parseAnalysisResult(content)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Category != "unknown" {
		t.Fatalf("expected category unknown, got %q", result.Category)
	}

	if result.Priority != "medium" {
		t.Fatalf("expected priority medium, got %q", result.Priority)
	}

	if result.Keywords == nil {
		t.Fatal("expected keywords to be initialized")
	}

	if len(result.Keywords) != 0 {
		t.Fatalf("expected empty keywords, got %#v", result.Keywords)
	}

	if !result.Escalate {
		t.Fatal("expected escalate to be true")
	}

	if result.Summary != "Нужна помощь" {
		t.Fatalf("expected summary to be preserved, got %q", result.Summary)
	}

	if result.Reason != "Сложный случай" {
		t.Fatalf("expected reason to be preserved, got %q", result.Reason)
	}

	if result.SuggestAction != "Проверить вручную" {
		t.Fatalf("expected suggest action to be preserved, got %q", result.SuggestAction)
	}
}

func TestAnalyze_ReturnsErrorWhenProviderReturnsNonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer server.Close()

	analyzer, err := NewAnalyzer(Config{
		APIKey:     "secret-token",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = analyzer.Analyze(context.Background(), model.AnalysisRequest{
		DialogID: "dlg_3",
		UserID:   "user_3",
		Message:  "Сообщение",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "got response code 502") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAnalyze_ReturnsErrorWhenResponseHasNoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()

	analyzer, err := NewAnalyzer(Config{
		APIKey:     "secret-token",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = analyzer.Analyze(context.Background(), model.AnalysisRequest{
		DialogID: "dlg_4",
		UserID:   "user_4",
		Message:  "Сообщение",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "response has no choices") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAnalyze_ReturnsErrorWhenProviderStopsForUnexpectedReason(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"finish_reason": "length",
					"message": {
						"role": "assistant",
						"content": "{\"category\":\"payment\",\"priority\":\"high\",\"keywords\":[\"оплата\"]}"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	analyzer, err := NewAnalyzer(Config{
		APIKey:     "secret-token",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = analyzer.Analyze(context.Background(), model.AnalysisRequest{
		DialogID: "dlg_5",
		UserID:   "user_5",
		Message:  "Сообщение",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), `deepseek finished with reason "length"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAnalyze_ReturnsErrorWhenResponseContentIsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"finish_reason": "stop",
					"message": {
						"role": "assistant",
						"content": "   "
					}
				}
			]
		}`))
	}))
	defer server.Close()

	analyzer, err := NewAnalyzer(Config{
		APIKey:     "secret-token",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = analyzer.Analyze(context.Background(), model.AnalysisRequest{
		DialogID: "dlg_6",
		UserID:   "user_6",
		Message:  "Сообщение",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "response content is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAnalyze_ReturnsErrorWhenModelResponseIsNotValidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"finish_reason": "stop",
					"message": {
						"role": "assistant",
						"content": "not json"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	analyzer, err := NewAnalyzer(Config{
		APIKey:     "secret-token",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = analyzer.Analyze(context.Background(), model.AnalysisRequest{
		DialogID: "dlg_7",
		UserID:   "user_7",
		Message:  "Сообщение",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "parse analysis result") {
		t.Fatalf("unexpected error: %v", err)
	}
}
