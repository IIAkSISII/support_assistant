package service

import (
	"context"
	"github.com/IIAkSISII/support-assistant/internal/model"
	"github.com/IIAkSISII/support-assistant/internal/repository/history"
	"github.com/IIAkSISII/support-assistant/internal/repository/knowledge"
)

const (
	defaultHistoryLimit = 10
	roleUser            = "user"
	roleBot             = "bot"
	defaultReply        = "Я передам ваше обращение оператору. Он изучит проблему и поможет вам."
)

type Analyzer interface {
	Analyze(ctx context.Context, request model.AnalysisRequest) (model.AnalysisResult, error)
}

type Processor interface {
	Process(ctx context.Context, incoming model.IncomingMessage) (model.ProcessResult, error)
}

type MessageProcessor struct {
	history      history.HistoryRepository
	knowledge    knowledge.KnowledgeRepository
	analyzer     Analyzer
	historyLimit int
}

func NewMessageProcessor(
	history history.HistoryRepository,
	knowledge knowledge.KnowledgeRepository,
	analyzer Analyzer,
	historyLimit int,
) *MessageProcessor {
	if historyLimit <= 0 {
		historyLimit = defaultHistoryLimit
	}
	return &MessageProcessor{
		history:      history,
		knowledge:    knowledge,
		analyzer:     analyzer,
		historyLimit: historyLimit,
	}
}

func (mp *MessageProcessor) Process(ctx context.Context, incoming model.IncomingMessage) (model.ProcessResult, error) {
	userMessage := model.Message{
		Role:    roleUser, // потом наверное лучше заменить на то, что роль приходит от СпросиИИ, а не хардкод
		Content: incoming.Message,
	}

	if err := mp.history.AddMessage(incoming.DialogID, userMessage); err != nil {
		return model.ProcessResult{}, err
	}

	history, err := mp.history.GetLastMessages(incoming.DialogID, mp.historyLimit)
	if err != nil {
		return model.ProcessResult{}, err
	}

	analysis, err := mp.analyzer.Analyze(ctx, model.AnalysisRequest{
		DialogID:         incoming.DialogID,
		UserID:           incoming.UserID,
		Message:          incoming.Message,
		History:          history,
		KnowledgeEntries: mp.knowledge.GetEntries(),
	})
	if err != nil {
		return model.ProcessResult{}, err
	}

	reply, found := mp.knowledge.FindAnswer(analysis.Category, analysis.Keywords)

	shouldEscalate := !found || analysis.Escalate

	if !found {
		reply = defaultReply
	}

	botMessage := model.Message{
		Role:    roleBot,
		Content: reply,
	}

	if err := mp.history.AddMessage(incoming.DialogID, botMessage); err != nil {
		return model.ProcessResult{}, err
	}

	result := model.ProcessResult{
		DialogID: incoming.DialogID,
		UserID:   incoming.UserID,
		Category: analysis.Category,
		Priority: analysis.Priority,
		Keywords: analysis.Keywords,
		Reply:    reply,
		Escalate: shouldEscalate,
	}
	if shouldEscalate {
		result.OperatorContext = &model.OperatorContext{
			Summary:       analysis.Summary,
			Reason:        analysis.Reason,
			SuggestAction: analysis.SuggestAction,
			DialogHistory: appendMessage(history, botMessage),
		}
	}
	return result, nil
}

func appendMessage(messages []model.Message, message model.Message) []model.Message {
	copied := make([]model.Message, 0, len(messages)+1)
	copied = append(copied, messages...)
	copied = append(copied, message)
	return copied
}
