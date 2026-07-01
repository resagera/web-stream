package chat

import "time"

type Message struct {
	ID        string    `json:"id"`
	GuestID   string    `json:"guest_id"`
	Name      string    `json:"name"`
	PhotoURL  string    `json:"photo_url,omitempty"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type Envelope struct {
	Type     string    `json:"type"`
	Message  *Message  `json:"message,omitempty"`
	Messages []Message `json:"messages,omitempty"`
	Error    string    `json:"error,omitempty"`
}
