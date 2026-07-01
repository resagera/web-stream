package journal

import "time"

type Entry struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	GuestID   string         `json:"guest_id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Role      string         `json:"role,omitempty"`
	IP        string         `json:"ip,omitempty"`
	Detail    string         `json:"detail,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type Event struct {
	Type     string
	GuestID  string
	Name     string
	Role     string
	IP       string
	Detail   string
	Metadata map[string]any
}
