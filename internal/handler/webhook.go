package handler

import (
	"context"
	"encoding/json"
	"github.com/IIAkSISII/support-assistant/internal/client"
	"github.com/IIAkSISII/support-assistant/internal/model"
	"github.com/IIAkSISII/support-assistant/internal/service"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type WebhookHandler struct {
	processor service.Processor
	logger    *slog.Logger
	sender    client.Sender
}

func NewWebhookHandler(processor service.Processor, logger ...*slog.Logger) *WebhookHandler {
	l := slog.Default()

	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}

	return &WebhookHandler{processor: processor, logger: l}
}

func NewWebhookHandlerWithSender(
	processor service.Processor,
	sender client.Sender,
	logger ...*slog.Logger,
) *WebhookHandler {
	handler := NewWebhookHandler(processor, logger...)
	handler.sender = sender

	return handler
}

// @Summary		Обработка webhook-запроса поддержки
// @Description	Принимает одно входящее событие webhook, классифицирует сообщение в поддержку, выбирает готовый ответ из базы знаний и при необходимости подготавливает контекст для эскалации оператору.
// @Tags			webhook
// @Accept			json
// @Produce		json
// @Param			request	body		WebhookRequest	true	"Тело входящего webhook-запроса"
// @Success		200		{object}	model.ProcessResult
// @Failure		400		{object}	ErrorResponse
// @Failure		405		{object}	ErrorResponse
// @Failure		500		{object}	ErrorResponse
// @Router			/webhook [post]
func (j *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		j.logger.Warn(
			"webhook rejected",
			"reason", "method not allowed",
			"method", r.Method,
			"path", r.URL.Path,
		)

		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error: "Only POST method is allowed",
		})
		return
	}

	var request WebhookRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		j.logger.Warn(
			"webhook rejected",
			"reason", "invalid json",
			"error", err.Error(),
		)

		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "Invalid JSON",
		})
		return
	}

	if request.Event != "message_created" {
		j.logger.Info(
			"webhook ignored",
			"reason", "unsupported event",
			"event", request.Event,
			"message_id", request.ID,
			"conversation_id", request.Conversation.ID,
		)

		writeJSON(w, http.StatusOK, IgnoredResponse{
			Status: "ignored",
			Reason: "Only 'message_created' event is allowed",
		})
		return
	}
	if request.MessageType != "incoming" {
		j.logger.Info(
			"webhook ignored",
			"reason", "unsupported message type",
			"message_type", request.MessageType,
			"message_id", request.ID,
			"conversation_id", request.Conversation.ID,
		)

		writeJSON(w, http.StatusOK, IgnoredResponse{
			Status: "ignored",
			Reason: "Unsupported message type",
		})
		return
	}
	if request.Private {
		j.logger.Info(
			"webhook ignored",
			"reason", "private message",
			"message_id", request.ID,
			"conversation_id", request.Conversation.ID,
		)

		writeJSON(w, http.StatusOK, IgnoredResponse{
			Status: "ignored",
			Reason: "Private message",
		})
		return
	}
	if strings.TrimSpace(request.Content) == "" {
		j.logger.Warn(
			"webhook rejected",
			"reason", "empty content",
			"message_id", request.ID,
			"conversation_id", request.Conversation.ID,
			"sender_id", request.Sender.ID,
		)

		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "Content is required",
		})
		return
	}
	if request.Conversation.ID == 0 {
		j.logger.Warn(
			"webhook rejected",
			"reason", "conversation id is required",
			"message_id", request.ID,
			"sender_id", request.Sender.ID,
		)

		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "conversation id is required",
		})
		return
	}
	if request.Sender.ID == 0 {
		j.logger.Warn(
			"webhook rejected",
			"reason", "sender id is required",
			"message_id", request.ID,
			"conversation_id", request.Conversation.ID,
		)

		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "sender id is required",
		})
		return
	}

	incoming := model.IncomingMessage{
		DialogID: strconv.FormatInt(request.Conversation.ID, 10),
		UserID:   strconv.FormatInt(request.Sender.ID, 10),
		Message:  strings.TrimSpace(request.Content),
	}

	j.logger.Info(
		"webhook accepted",
		"message_id", request.ID,
		"dialog_id", incoming.DialogID,
		"user_id", incoming.UserID,
		"inbox_id", request.Inbox.ID,
		"inbox_name", request.Inbox.Name,
		"content_length", len(request.Content),
		"attachments_count", len(request.Attachments),
	)

	result, err := j.processor.Process(r.Context(), incoming)
	if err != nil {
		j.logger.Error(
			"webhook processing failed",
			"error", err.Error(),
			"message_id", request.ID,
			"dialog_id", incoming.DialogID,
			"user_id", incoming.UserID,
		)

		writeJSON(w, http.StatusInternalServerError, ErrorResponse{
			Error: "Processing message failed",
		})
		return
	}

	j.logger.Info(
		"webhook processed",
		"message_id", request.ID,
		"dialog_id", result.DialogID,
		"user_id", result.UserID,
		"category", result.Category,
		"priority", result.Priority,
		"keywords", result.Keywords,
		"escalate", result.Escalate,
	)

	j.sendProcessResultToExternalAPI(r.Context(), request, result)

	writeJSON(w, http.StatusOK, result)
}

func (j *WebhookHandler) sendProcessResultToExternalAPI(
	ctx context.Context,
	request WebhookRequest,
	result model.ProcessResult,
) {
	if j.sender == nil {
		return
	}

	accountID := request.Account.ID
	if accountID == 0 {
		j.logger.Warn(
			"process result delivery skipped",
			"reason", "account id is missing",
			"message_id", request.ID,
			"conversation_id", request.Conversation.ID,
		)
		return
	}

	sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := j.sender.SendProcessResult(sendCtx, accountID, request.Conversation.ID, result); err != nil {
		j.logger.Error(
			"process result delivery failed",
			"error", err.Error(),
			"message_id", request.ID,
			"account_id", accountID,
			"conversation_id", request.Conversation.ID,
		)
		return
	}

	j.logger.Info(
		"process result delivered",
		"message_id", request.ID,
		"account_id", accountID,
		"conversation_id", request.Conversation.ID,
	)
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(data)
}
