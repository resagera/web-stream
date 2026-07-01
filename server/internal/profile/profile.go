package profile

import "time"

const (
	RoleGuest       = "guest"
	RoleBroadcaster = "broadcaster"
	RoleAdmin       = "admin"
)

type Guest struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Name      string    `json:"name"`
	PhotoURL  string    `json:"photo_url,omitempty"`
	Secret    string    `json:"secret"`
	IP        string    `json:"ip,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NormalizeRole(role string) string {
	switch role {
	case RoleBroadcaster:
		return RoleBroadcaster
	case RoleAdmin:
		return RoleAdmin
	default:
		return RoleGuest
	}
}

func CanPublish(role string) bool {
	role = NormalizeRole(role)
	return role == RoleBroadcaster || role == RoleAdmin
}
