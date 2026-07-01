package event

import "time"

type Settings struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func DefaultSettings() Settings {
	return Settings{
		Title:       "Family Stream",
		Description: "",
		UpdatedAt:   time.Now().UTC(),
	}
}
