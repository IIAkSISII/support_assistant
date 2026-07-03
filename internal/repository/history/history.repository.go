package history

import (
	"github.com/IIAkSISII/support-assistant/internal/model"
	"sync"
)

type HistoryRepository interface {
	AddMessage(dialogID string, message model.Message) error
	GetLastMessages(dialogID string, limit int) ([]model.Message, error)
}

type historyRepository struct {
	mutex   sync.RWMutex
	dialogs map[string][]model.Message
}

func NewHistoryRepository() HistoryRepository {
	return &historyRepository{
		dialogs: make(map[string][]model.Message),
	}
}

func (r *historyRepository) AddMessage(dialogID string, message model.Message) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.dialogs[dialogID] = append(r.dialogs[dialogID], message)

	return nil
}

func (r *historyRepository) GetLastMessages(dialogID string, limit int) ([]model.Message, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	messages := r.dialogs[dialogID]

	if limit <= 0 {
		return []model.Message{}, nil
	}
	if len(messages) <= limit {
		copied := make([]model.Message, len(messages))
		copy(copied, messages)
		return copied, nil
	}
	last := messages[len(messages)-limit:]
	copied := make([]model.Message, len(last))
	copy(copied, last)

	return copied, nil
}
