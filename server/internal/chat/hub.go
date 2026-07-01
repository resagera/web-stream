package chat

import (
	"sync"
	"time"

	"home-stream/server/internal/id"
	"home-stream/server/internal/profile"
)

type Client interface {
	Send(Envelope) error
	Close() error
}

type Presence struct {
	GuestID     string    `json:"guest_id"`
	Name        string    `json:"name"`
	Role        string    `json:"role"`
	PhotoURL    string    `json:"photo_url,omitempty"`
	ConnectedAt time.Time `json:"connected_at"`
}

type Hub struct {
	mu      sync.RWMutex
	limit   int
	storage Storage
	history []Message
	clients map[Client]Presence
}

func NewHub(limit int, storage Storage) (*Hub, error) {
	history, err := storage.LoadLast(limit)
	if err != nil {
		return nil, err
	}
	return &Hub{
		limit:   limit,
		storage: storage,
		history: history,
		clients: make(map[Client]Presence),
	}, nil
}

func (h *Hub) Add(client Client, guest profile.Guest) error {
	h.mu.Lock()
	h.clients[client] = Presence{
		GuestID:     guest.ID,
		Name:        guest.Name,
		Role:        profile.NormalizeRole(guest.Role),
		PhotoURL:    guest.PhotoURL,
		ConnectedAt: time.Now().UTC(),
	}
	history := append([]Message(nil), h.history...)
	h.mu.Unlock()

	return client.Send(Envelope{Type: "history", Messages: history})
}

func (h *Hub) Remove(client Client) {
	h.mu.Lock()
	delete(h.clients, client)
	h.mu.Unlock()
	_ = client.Close()
}

func (h *Hub) Presence() []Presence {
	h.mu.RLock()
	defer h.mu.RUnlock()

	presence := make([]Presence, 0, len(h.clients))
	for _, item := range h.clients {
		presence = append(presence, item)
	}
	return presence
}

func (h *Hub) History() []Message {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return append([]Message(nil), h.history...)
}

func (h *Hub) Post(guest profile.Guest, text string) (Message, error) {
	messageID, err := id.New(16)
	if err != nil {
		return Message{}, err
	}

	message := Message{
		ID:        messageID,
		GuestID:   guest.ID,
		Name:      guest.Name,
		PhotoURL:  guest.PhotoURL,
		Text:      text,
		CreatedAt: time.Now().UTC(),
	}

	if err := h.storage.Append(message); err != nil {
		return Message{}, err
	}

	h.mu.Lock()
	h.history = append(h.history, message)
	if len(h.history) > h.limit {
		h.history = h.history[len(h.history)-h.limit:]
	}
	clients := make([]Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.Unlock()

	envelope := Envelope{Type: "message", Message: &message}
	for _, client := range clients {
		if err := client.Send(envelope); err != nil {
			h.Remove(client)
		}
	}

	return message, nil
}
