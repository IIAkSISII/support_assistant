package handler

type WebhookRequest struct {
	Event        string              `json:"event" example:"message_created"`
	ID           int64               `json:"id" example:"123"`
	Content      string              `json:"content" example:"Я оплатил подписку, но доступ не появился"`
	MessageType  string              `json:"message_type" example:"incoming"`
	Private      bool                `json:"private" example:"false"`
	Sender       WebhookSender       `json:"sender"`
	Conversation WebhookConversation `json:"conversation"`
	Inbox        WebhookInbox        `json:"inbox"`
	Attachments  []WebhookAttachment `json:"attachments"`
}

type WebhookSender struct {
	ID    int64  `json:"id" example:"456"`
	Name  string `json:"name" example:"Anton"`
	Email string `json:"email" example:"anton88@example.com"`
	Type  string `json:"type" example:"user"`
}

type WebhookConversation struct {
	ID       int64  `json:"id" example:"789"`
	Status   string `json:"status" example:"open"`
	Priority string `json:"priority" example:"medium"`
}

type WebhookInbox struct {
	ID   int64  `json:"id" example:"10"`
	Name string `json:"name" example:"Support"`
}

type WebhookAttachment struct {
	ID       int64  `json:"id" example:"1001"`
	FileURL  string `json:"file_url" example:"https://example.com/files/1001.png"`
	FileType string `json:"file_type" example:"image/png"`
	ThumbURL string `json:"thumb_url" example:"https://example.com/files/1001-thumb.png"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"Invalid JSON"`
}

type IgnoredResponse struct {
	Status string `json:"status" example:"ignored"`
	Reason string `json:"reason" example:"Only 'message_created' event is allowed"`
}
