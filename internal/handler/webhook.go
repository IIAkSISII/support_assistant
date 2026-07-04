package handler

import (
	"encoding/json"
	"github.com/IIAkSISII/support-assistant/internal/model"
	"github.com/IIAkSISII/support-assistant/internal/service"
	"net/http"
	"strconv"
	"strings"
)

type WebhookHandler struct {
	processor service.Processor
}

func NewWebhookHandler(processor service.Processor) *WebhookHandler {
	return &WebhookHandler{processor: processor}
}

func (j *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Error: "Only POST method is allowed",
		})
		return
	}

	var request webhookRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "Invalid JSON",
		})
		return
	}

	if request.Event != "message_created" {
		writeJSON(w, http.StatusOK, ignoredResponse{
			Status: "ignored",
			Reason: "Only 'message_created' event is allowed",
		})
		return
	}
	if request.MessageType != "incoming" {
		writeJSON(w, http.StatusOK, ignoredResponse{
			Status: "ignored",
			Reason: "Unsupported message type",
		})
		return
	}
	if request.Private {
		writeJSON(w, http.StatusOK, ignoredResponse{
			Status: "ignored",
			Reason: "Private message",
		})
		return
	}
	if strings.TrimSpace(request.Content) == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "Content is required",
		})
		return
	}
	if request.Conversation.ID == 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "conversation id is required",
		})
		return
	}
	if request.Sender.ID == 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "sender id is required",
		})
		return
	}

	incoming := model.IncomingMessage{
		DialogID: strconv.FormatInt(request.Conversation.ID, 10),
		UserID:   strconv.FormatInt(request.Sender.ID, 10),
		Message:  strings.TrimSpace(request.Content),
	}

	result, err := j.processor.Process(r.Context(), incoming)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Error: "Processing message failed",
		})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(data)
}
