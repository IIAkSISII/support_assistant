package history

import (
	"github.com/IIAkSISII/support-assistant/internal/model"
	"testing"
)

func TestHistoryRepo_AddMessage_GetMessage(t *testing.T) {
	repo := NewHistoryRepository()

	err := repo.AddMessage("dig_1", model.Message{
		Role:    "user",
		Content: "Не могу войти",
	})
	if err != nil {
		t.Fatalf("expeted no error, got %v", err)
	}
	message, err := repo.GetLastMessages("dig_1", 10)
	if err != nil {
		t.Fatalf("expeted no error, got %v", err)
	}

	if len(message) != 1 {
		t.Fatalf("expeted len 1, got %v", len(message))
	}
	if message[0].Role != "user" {
		t.Fatalf("expected role user, got %v", message[0].Role)
	}
	if message[0].Content != "Не могу войти" {
		t.Errorf("expected content 'Не могу войти', got %s", message[0].Content)
	}
}

func TestHistoryStorage_GetLastMessagesWithLimit(t *testing.T) {
	repo := NewHistoryRepository()

	_ = repo.AddMessage("dlg_1", model.Message{Role: "user", Content: "one"})
	_ = repo.AddMessage("dlg_1", model.Message{Role: "bot", Content: "two"})
	_ = repo.AddMessage("dlg_1", model.Message{Role: "user", Content: "three"})

	messages, err := repo.GetLastMessages("dlg_1", 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	if messages[0].Content != "two" {
		t.Errorf("expected first message 'two', got %s", messages[0].Content)
	}

	if messages[1].Content != "three" {
		t.Errorf("expected second message 'three', got %s", messages[1].Content)
	}
}
