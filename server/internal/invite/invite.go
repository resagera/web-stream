package invite

import "time"

type Invite struct {
	Token     string    `json:"token"`
	Role      string    `json:"role"`
	Label     string    `json:"label,omitempty"`
	Active    bool      `json:"active"`
	MaxUses   int       `json:"max_uses"`
	UsedCount int       `json:"used_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (i Invite) CanUse() bool {
	return i.Active && (i.MaxUses == 0 || i.UsedCount < i.MaxUses)
}
