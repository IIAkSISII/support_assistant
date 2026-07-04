package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IIAkSISII/support-assistant/internal/model"
)

type fakeProcessor struct {
	result model.ProcessResult
	err    error

	lastCtx      context.Context
	lastIncoming model.IncomingMessage
	called       bool
}

func (p *fakeProcessor) Process(ctx context.Context, incoming model.IncomingMessage) (model.ProcessResult, error) {
	p.called = true
	p.lastCtx = ctx
	p.lastIncoming = incoming

	if p.err != nil {
		return model.ProcessResult{}, p.err
	}

	return p.result, nil
}

func TestWebhookHandler_ProcessesIncomingMessage(t *testing.T) {
	processor := &fakeProcessor{
		result: model.ProcessResult{
			DialogID: "789",
			UserID:   "456",
			Category: "auth",
			Priority: "medium",
			Keywords: []string{"логин"},
			Reply:    "Попробуйте восстановить пароль через страницу входа.",
			Escalate: false,
		},
	}

	handler := NewWebhookHandler(processor)

	body := `{
		"event": "message_created",
		"id": 123,
		"content": "Не могу войти в аккаунт",
		"message_type": "incoming",
		"private": false,
		"sender": {
			"id": 456,
			"name": "Антон",
			"email": "anton88@example.com",
			"type": "user"
		},
		"conversation": {
			"id": 789,
			"status": "open",
			"priority": "medium"
		},
		"inbox": {
			"id": 10,
			"name": "Support"
		},
		"attachments": []
	}`

	request := httptest.NewRequest(http.MethodPost, "/webhook/support", strings.NewReader(body))
	response := httptest.NewRecorder()

	handler.HandleWebhook(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	if !processor.called {
		t.Fatal("expected processor to be called")
	}

	if processor.lastCtx != request.Context() {
		t.Error("expected request context to be passed to processor")
	}

	expectedIncoming := model.IncomingMessage{
		DialogID: "789",
		UserID:   "456",
		Message:  "Не могу войти в аккаунт",
	}

	if processor.lastIncoming != expectedIncoming {
		t.Errorf("unexpected incoming message: got %#v, want %#v", processor.lastIncoming, expectedIncoming)
	}

	var result model.ProcessResult

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.DialogID != processor.result.DialogID {
		t.Errorf("expected dialog id %q, got %q", processor.result.DialogID, result.DialogID)
	}

	if result.UserID != processor.result.UserID {
		t.Errorf("expected user id %q, got %q", processor.result.UserID, result.UserID)
	}

	if result.Category != processor.result.Category {
		t.Errorf("expected category %q, got %q", processor.result.Category, result.Category)
	}

	if result.Priority != processor.result.Priority {
		t.Errorf("expected priority %q, got %q", processor.result.Priority, result.Priority)
	}

	if result.Reply != processor.result.Reply {
		t.Errorf("expected reply %q, got %q", processor.result.Reply, result.Reply)
	}

	if result.Escalate != processor.result.Escalate {
		t.Errorf("expected escalate %v, got %v", processor.result.Escalate, result.Escalate)
	}
}

func TestWebhookHandler_TrimsContentBeforeProcessing(t *testing.T) {
	processor := &fakeProcessor{
		result: model.ProcessResult{
			DialogID: "789",
			UserID:   "456",
			Reply:    "ok",
		},
	}

	handler := NewWebhookHandler(processor)

	body := `{
		"event": "message_created",
		"content": "   Не могу войти   ",
		"message_type": "incoming",
		"private": false,
		"sender": {
			"id": 456
		},
		"conversation": {
			"id": 789
		}
	}`

	request := httptest.NewRequest(http.MethodPost, "/webhook/support", strings.NewReader(body))
	response := httptest.NewRecorder()

	handler.HandleWebhook(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	if !processor.called {
		t.Fatal("expected processor to be called")
	}

	if processor.lastIncoming.Message != "Не могу войти" {
		t.Errorf("expected trimmed message %q, got %q", "Не могу войти", processor.lastIncoming.Message)
	}
}

func TestWebhookHandler_IgnoresUnsupportedEvent(t *testing.T) {
	processor := &fakeProcessor{}
	handler := NewWebhookHandler(processor)

	body := `{
		"event": "conversation_updated",
		"content": "Не могу войти",
		"message_type": "incoming",
		"private": false,
		"sender": {
			"id": 456
		},
		"conversation": {
			"id": 789
		}
	}`

	request := httptest.NewRequest(http.MethodPost, "/webhook/support", strings.NewReader(body))
	response := httptest.NewRecorder()

	handler.HandleWebhook(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	if processor.called {
		t.Fatal("expected processor not to be called")
	}

	var result struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Status != "ignored" {
		t.Errorf("expected status %q, got %q", "ignored", result.Status)
	}

	if result.Reason != "Only 'message_created' event is allowed" {
		t.Errorf("unexpected reason: %q", result.Reason)
	}
}

func TestWebhookHandler_IgnoresOutgoingMessage(t *testing.T) {
	processor := &fakeProcessor{}
	handler := NewWebhookHandler(processor)

	body := `{
		"event": "message_created",
		"content": "Ответ бота",
		"message_type": "outgoing",
		"private": false,
		"sender": {
			"id": 456
		},
		"conversation": {
			"id": 789
		}
	}`

	request := httptest.NewRequest(http.MethodPost, "/webhook/support", strings.NewReader(body))
	response := httptest.NewRecorder()

	handler.HandleWebhook(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	if processor.called {
		t.Fatal("expected processor not to be called")
	}

	var result struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Status != "ignored" {
		t.Errorf("expected status %q, got %q", "ignored", result.Status)
	}

	if result.Reason != "Unsupported message type" {
		t.Errorf("unexpected reason: %q", result.Reason)
	}
}

func TestWebhookHandler_IgnoresPrivateMessage(t *testing.T) {
	processor := &fakeProcessor{}
	handler := NewWebhookHandler(processor)

	body := `{
		"event": "message_created",
		"content": "Внутренняя заметка",
		"message_type": "incoming",
		"private": true,
		"sender": {
			"id": 456
		},
		"conversation": {
			"id": 789
		}
	}`

	request := httptest.NewRequest(http.MethodPost, "/webhook/support", strings.NewReader(body))
	response := httptest.NewRecorder()

	handler.HandleWebhook(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	if processor.called {
		t.Fatal("expected processor not to be called")
	}

	var result struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Status != "ignored" {
		t.Errorf("expected status %q, got %q", "ignored", result.Status)
	}

	if result.Reason != "Private message" {
		t.Errorf("unexpected reason: %q", result.Reason)
	}
}

func TestWebhookHandler_ReturnsBadRequestForInvalidJSON(t *testing.T) {
	processor := &fakeProcessor{}
	handler := NewWebhookHandler(processor)

	request := httptest.NewRequest(http.MethodPost, "/webhook/support", strings.NewReader(`{invalid json`))
	response := httptest.NewRecorder()

	handler.HandleWebhook(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}

	if processor.called {
		t.Fatal("expected processor not to be called")
	}

	var result struct {
		Error string `json:"error"`
	}

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Error != "Invalid JSON" {
		t.Errorf("expected error %q, got %q", "Invalid JSON", result.Error)
	}
}

func TestWebhookHandler_ReturnsBadRequestForEmptyContent(t *testing.T) {
	processor := &fakeProcessor{}
	handler := NewWebhookHandler(processor)

	body := `{
		"event": "message_created",
		"content": "",
		"message_type": "incoming",
		"private": false,
		"sender": {
			"id": 456
		},
		"conversation": {
			"id": 789
		}
	}`

	request := httptest.NewRequest(http.MethodPost, "/webhook/support", strings.NewReader(body))
	response := httptest.NewRecorder()

	handler.HandleWebhook(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}

	if processor.called {
		t.Fatal("expected processor not to be called")
	}

	var result struct {
		Error string `json:"error"`
	}

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Error != "Content is required" {
		t.Errorf("expected error %q, got %q", "Content is required", result.Error)
	}
}

func TestWebhookHandler_ReturnsBadRequestForBlankContent(t *testing.T) {
	processor := &fakeProcessor{}
	handler := NewWebhookHandler(processor)

	body := `{
		"event": "message_created",
		"content": "     ",
		"message_type": "incoming",
		"private": false,
		"sender": {
			"id": 456
		},
		"conversation": {
			"id": 789
		}
	}`

	request := httptest.NewRequest(http.MethodPost, "/webhook/support", strings.NewReader(body))
	response := httptest.NewRecorder()

	handler.HandleWebhook(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}

	if processor.called {
		t.Fatal("expected processor not to be called")
	}
}

func TestWebhookHandler_ReturnsBadRequestForMissingConversationID(t *testing.T) {
	processor := &fakeProcessor{}
	handler := NewWebhookHandler(processor)

	body := `{
		"event": "message_created",
		"content": "Не могу войти",
		"message_type": "incoming",
		"private": false,
		"sender": {
			"id": 456
		},
		"conversation": {
			"id": 0
		}
	}`

	request := httptest.NewRequest(http.MethodPost, "/webhook/support", strings.NewReader(body))
	response := httptest.NewRecorder()

	handler.HandleWebhook(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}

	if processor.called {
		t.Fatal("expected processor not to be called")
	}

	var result struct {
		Error string `json:"error"`
	}

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Error != "conversation id is required" {
		t.Errorf("expected error %q, got %q", "conversation id is required", result.Error)
	}
}

func TestWebhookHandler_ReturnsBadRequestForMissingSenderID(t *testing.T) {
	processor := &fakeProcessor{}
	handler := NewWebhookHandler(processor)

	body := `{
		"event": "message_created",
		"content": "Не могу войти",
		"message_type": "incoming",
		"private": false,
		"sender": {
			"id": 0
		},
		"conversation": {
			"id": 789
		}
	}`

	request := httptest.NewRequest(http.MethodPost, "/webhook/support", strings.NewReader(body))
	response := httptest.NewRecorder()

	handler.HandleWebhook(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}

	if processor.called {
		t.Fatal("expected processor not to be called")
	}

	var result struct {
		Error string `json:"error"`
	}

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Error != "sender id is required" {
		t.Errorf("expected error %q, got %q", "sender id is required", result.Error)
	}
}

func TestWebhookHandler_ReturnsInternalServerErrorWhenProcessorFails(t *testing.T) {
	processor := &fakeProcessor{
		err: errors.New("processor failed"),
	}

	handler := NewWebhookHandler(processor)

	body := `{
		"event": "message_created",
		"content": "Не могу войти",
		"message_type": "incoming",
		"private": false,
		"sender": {
			"id": 456
		},
		"conversation": {
			"id": 789
		}
	}`

	request := httptest.NewRequest(http.MethodPost, "/webhook/support", strings.NewReader(body))
	response := httptest.NewRecorder()

	handler.HandleWebhook(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", response.Code)
	}

	if !processor.called {
		t.Fatal("expected processor to be called")
	}

	var result struct {
		Error string `json:"error"`
	}

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Error != "Processing message failed" {
		t.Errorf("expected error %q, got %q", "Processing message failed", result.Error)
	}
}

func TestWebhookHandler_ReturnsMethodNotAllowedForGetRequest(t *testing.T) {
	processor := &fakeProcessor{}
	handler := NewWebhookHandler(processor)

	request := httptest.NewRequest(http.MethodGet, "/webhook/support", nil)
	response := httptest.NewRecorder()

	handler.HandleWebhook(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", response.Code)
	}

	if processor.called {
		t.Fatal("expected processor not to be called")
	}

	if response.Header().Get("Allow") != http.MethodPost {
		t.Errorf("expected Allow header %q, got %q", http.MethodPost, response.Header().Get("Allow"))
	}

	var result struct {
		Error string `json:"error"`
	}

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Error != "Only POST method is allowed" {
		t.Errorf("expected error %q, got %q", "Only POST method is allowed", result.Error)
	}
}
