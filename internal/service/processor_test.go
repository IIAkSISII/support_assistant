package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/IIAkSISII/support-assistant/internal/model"
)

type fakeHistoryRepository struct {
	messages []model.Message
	addErr   error
	getErr   error
}

func (r *fakeHistoryRepository) AddMessage(dialogID string, message model.Message) error {
	if r.addErr != nil {
		return r.addErr
	}

	r.messages = append(r.messages, message)
	return nil
}

func (r *fakeHistoryRepository) GetLastMessages(dialogID string, limit int) ([]model.Message, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}

	if limit <= 0 {
		return []model.Message{}, nil
	}

	if len(r.messages) <= limit {
		copied := make([]model.Message, len(r.messages))
		copy(copied, r.messages)
		return copied, nil
	}

	last := r.messages[len(r.messages)-limit:]

	copied := make([]model.Message, len(last))
	copy(copied, last)

	return copied, nil
}

type fakeKnowledgeRepository struct {
	answer  string
	found   bool
	entries []model.Entry

	lastCategory string
	lastKeywords []string
}

func (r *fakeKnowledgeRepository) FindAnswer(category string, keywords []string) (string, bool) {
	r.lastCategory = category
	r.lastKeywords = append([]string(nil), keywords...)

	return r.answer, r.found
}

func (r *fakeKnowledgeRepository) GetEntries() []model.Entry {
	entries := make([]model.Entry, len(r.entries))

	for i, entry := range r.entries {
		entries[i] = model.Entry{
			Category: entry.Category,
			Keywords: append([]string(nil), entry.Keywords...),
			Answer:   entry.Answer,
		}
	}

	return entries
}

type fakeAnalyzer struct {
	result model.AnalysisResult
	err    error

	lastRequest model.AnalysisRequest
}

func (a *fakeAnalyzer) Analyze(ctx context.Context, request model.AnalysisRequest) (model.AnalysisResult, error) {
	a.lastRequest = request

	if a.err != nil {
		return model.AnalysisResult{}, a.err
	}

	return a.result, nil
}

func TestProcessor_UsesKnowledgeBaseAnswer(t *testing.T) {
	historyRepo := &fakeHistoryRepository{}

	knowledgeEntries := []model.Entry{
		{
			Category: "payment",
			Keywords: []string{"оплата", "подписка"},
			Answer:   "Ответ из базы знаний.",
		},
	}

	knowledgeRepo := &fakeKnowledgeRepository{
		answer:  "Ответ из базы знаний.",
		found:   true,
		entries: knowledgeEntries,
	}

	analyzer := &fakeAnalyzer{
		result: model.AnalysisResult{
			Category: "payment",
			Priority: "high",
			Keywords: []string{"оплата", "подписка"},
			Escalate: false,
			Summary:  "Пользователь оплатил подписку, но доступ не появился.",
		},
	}

	processor := NewMessageProcessor(historyRepo, knowledgeRepo, analyzer, 10)

	result, err := processor.Process(context.Background(), model.IncomingMessage{
		DialogID: "789",
		UserID:   "456",
		Message:  "Я оплатил подписку, но доступ не появился",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Reply != knowledgeRepo.answer {
		t.Errorf("expected knowledge base reply, got %s", result.Reply)
	}

	if result.Escalate {
		t.Error("expected escalate to be false")
	}

	if result.OperatorContext != nil {
		t.Error("expected operator context to be nil")
	}

	if knowledgeRepo.lastCategory != "payment" {
		t.Errorf("expected knowledge category payment, got %s", knowledgeRepo.lastCategory)
	}

	if !reflect.DeepEqual(knowledgeRepo.lastKeywords, []string{"оплата", "подписка"}) {
		t.Errorf("unexpected knowledge keywords: %#v", knowledgeRepo.lastKeywords)
	}

	if len(historyRepo.messages) != 2 {
		t.Fatalf("expected 2 messages in history, got %d", len(historyRepo.messages))
	}

	if historyRepo.messages[0] != (model.Message{Role: roleUser, Content: "Я оплатил подписку, но доступ не появился"}) {
		t.Errorf("unexpected user message: %#v", historyRepo.messages[0])
	}

	if historyRepo.messages[1] != (model.Message{Role: roleBot, Content: knowledgeRepo.answer}) {
		t.Errorf("unexpected bot message: %#v", historyRepo.messages[1])
	}

	if analyzer.lastRequest.DialogID != "789" {
		t.Errorf("expected analyzer dialog id 789, got %s", analyzer.lastRequest.DialogID)
	}

	if analyzer.lastRequest.UserID != "456" {
		t.Errorf("expected analyzer user id 456, got %s", analyzer.lastRequest.UserID)
	}

	if analyzer.lastRequest.Message != "Я оплатил подписку, но доступ не появился" {
		t.Errorf("unexpected analyzer message: %s", analyzer.lastRequest.Message)
	}

	if !reflect.DeepEqual(analyzer.lastRequest.KnowledgeEntries, knowledgeEntries) {
		t.Errorf("unexpected knowledge entries: %#v", analyzer.lastRequest.KnowledgeEntries)
	}

	if len(analyzer.lastRequest.History) != 1 {
		t.Fatalf("expected 1 message in analyzer history, got %d", len(analyzer.lastRequest.History))
	}

	if analyzer.lastRequest.History[0].Role != roleUser {
		t.Errorf("expected analyzer history role user, got %s", analyzer.lastRequest.History[0].Role)
	}
}

func TestProcessor_EscalatesWhenKnowledgeBaseAnswerNotFound(t *testing.T) {
	historyRepo := &fakeHistoryRepository{}

	knowledgeRepo := &fakeKnowledgeRepository{
		found: false,
	}

	analyzer := &fakeAnalyzer{
		result: model.AnalysisResult{
			Category:      "payment",
			Priority:      "high",
			Keywords:      []string{"оплата", "подписка"},
			Escalate:      false,
			Summary:       "Пользователь оплатил подписку, но доступ не появился.",
			Reason:        "В базе знаний нет подходящего готового ответа.",
			SuggestAction: "Проверить платеж пользователя и статус подписки.",
		},
	}

	processor := NewMessageProcessor(historyRepo, knowledgeRepo, analyzer, 10)

	result, err := processor.Process(context.Background(), model.IncomingMessage{
		DialogID: "789",
		UserID:   "456",
		Message:  "Я оплатил подписку, но доступ не появился",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !result.Escalate {
		t.Error("expected escalate to be true")
	}

	if result.Reply != defaultReply {
		t.Errorf("expected default reply, got %s", result.Reply)
	}

	if result.OperatorContext == nil {
		t.Fatal("expected operator context")
	}

	if result.OperatorContext.Summary != analyzer.result.Summary {
		t.Errorf("expected summary %q, got %q", analyzer.result.Summary, result.OperatorContext.Summary)
	}

	if result.OperatorContext.Reason != analyzer.result.Reason {
		t.Errorf("expected reason %q, got %q", analyzer.result.Reason, result.OperatorContext.Reason)
	}

	if result.OperatorContext.SuggestAction != analyzer.result.SuggestAction {
		t.Errorf("expected suggest action %q, got %q", analyzer.result.SuggestAction, result.OperatorContext.SuggestAction)
	}

	if len(result.OperatorContext.DialogHistory) != 2 {
		t.Fatalf("expected 2 messages in operator history, got %d", len(result.OperatorContext.DialogHistory))
	}

	if result.OperatorContext.DialogHistory[0].Role != roleUser {
		t.Errorf("expected first operator history role user, got %s", result.OperatorContext.DialogHistory[0].Role)
	}

	if result.OperatorContext.DialogHistory[1].Role != roleBot {
		t.Errorf("expected second operator history role bot, got %s", result.OperatorContext.DialogHistory[1].Role)
	}
}

func TestProcessor_EscalatesWhenAnalyzerRequiresEscalation(t *testing.T) {
	historyRepo := &fakeHistoryRepository{}

	knowledgeRepo := &fakeKnowledgeRepository{
		answer: "Готовый ответ из базы знаний.",
		found:  true,
	}

	analyzer := &fakeAnalyzer{
		result: model.AnalysisResult{
			Category:      "bug",
			Priority:      "high",
			Keywords:      []string{"ошибка"},
			Escalate:      true,
			Summary:       "У пользователя сложная техническая ошибка.",
			Reason:        "Требуется проверка оператором.",
			SuggestAction: "Передать обращение техническому специалисту.",
		},
	}

	processor := NewMessageProcessor(historyRepo, knowledgeRepo, analyzer, 10)

	result, err := processor.Process(context.Background(), model.IncomingMessage{
		DialogID: "dlg_1",
		UserID:   "user_1",
		Message:  "У меня сложная ошибка",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !result.Escalate {
		t.Error("expected escalate to be true")
	}

	if result.Reply != knowledgeRepo.answer {
		t.Errorf("expected knowledge base reply, got %s", result.Reply)
	}

	if result.OperatorContext == nil {
		t.Fatal("expected operator context")
	}

	if result.OperatorContext.Summary != analyzer.result.Summary {
		t.Errorf("expected summary %q, got %q", analyzer.result.Summary, result.OperatorContext.Summary)
	}
}

func TestProcessor_ReturnsErrorWhenAnalyzerFails(t *testing.T) {
	historyRepo := &fakeHistoryRepository{}

	knowledgeRepo := &fakeKnowledgeRepository{
		answer: "Готовый ответ из базы знаний.",
		found:  true,
	}

	analyzerErr := errors.New("analyzer failed")

	analyzer := &fakeAnalyzer{
		err: analyzerErr,
	}

	processor := NewMessageProcessor(historyRepo, knowledgeRepo, analyzer, 10)

	_, err := processor.Process(context.Background(), model.IncomingMessage{
		DialogID: "dlg_1",
		UserID:   "user_1",
		Message:  "Не могу войти",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, analyzerErr) {
		t.Errorf("expected analyzer error, got %v", err)
	}

	if len(historyRepo.messages) != 1 {
		t.Fatalf("expected only user message in history, got %d messages", len(historyRepo.messages))
	}

	if historyRepo.messages[0].Role != roleUser {
		t.Errorf("expected saved message role user, got %s", historyRepo.messages[0].Role)
	}
}
