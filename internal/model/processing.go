package model

type IncomingMessage struct {
	DialogID string `json:"dialog_id"`
	UserID   string `json:"user_id"`
	Message  string `json:"message"`
}

type AnalysisRequest struct {
	DialogID         string    `json:"dialog_id"`
	UserID           string    `json:"user_id"`
	Message          string    `json:"message"`
	History          []Message `json:"history"`
	KnowledgeEntries []Entry   `json:"knowledge_entries"`
}

type AnalysisResult struct {
	Category      string   `json:"category"`
	Priority      string   `json:"priority"`
	Keywords      []string `json:"keywords"`
	Escalate      bool     `json:"escalate"`
	Summary       string   `json:"summary"`
	Reason        string   `json:"reason"`
	SuggestAction string   `json:"suggest_action"`
}

type OperatorContext struct {
	Summary       string    `json:"summary"`
	Reason        string    `json:"reason"`
	SuggestAction string    `json:"suggest_action"`
	DialogHistory []Message `json:"dialog_history"`
}

type ProcessResult struct {
	DialogID        string           `json:"dialog_id"`
	UserID          string           `json:"user_id"`
	Category        string           `json:"category"`
	Priority        string           `json:"priority"`
	Keywords        []string         `json:"keywords"`
	Reply           string           `json:"reply"`
	Escalate        bool             `json:"escalate"`
	OperatorContext *OperatorContext `json:"operator_context,omitempty"`
}
