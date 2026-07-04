package handler

type webhookRequest struct {
	Event        string              `json:"event"`
	ID           int64               `json:"id"`
	Content      string              `json:"content"`
	MessageType  string              `json:"message_type"`
	Private      bool                `json:"private"`
	Sender       webhookSender       `json:"sender"`
	Conversation webhookConversation `json:"conversation"`
	Inbox        webhookInbox        `json:"inbox"`
	Attachments  []webhookAttachment `json:"attachments"`
}

type webhookSender struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Type  string `json:"type"`
}

type webhookConversation struct {
	ID       int64  `json:"id"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
}

type webhookInbox struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type webhookAttachment struct {
	ID       int64  `json:"id"`
	FileURL  string `json:"file_url"`
	FileType string `json:"file_type"`
	ThumbURL string `json:"thumb_url"`
}

type errorResponse struct {
	Error string `json:"error"`
}
type ignoredResponse struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}
