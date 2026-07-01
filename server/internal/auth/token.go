package auth

import (
	"strings"

	"home-stream/server/internal/profile"
)

type Token struct {
	GuestID string
	Secret  string
}

func Issue(guest profile.Guest) string {
	return guest.ID + "." + guest.Secret
}

func Parse(raw string) (Token, bool) {
	guestID, secret, ok := strings.Cut(strings.TrimSpace(raw), ".")
	if !ok || guestID == "" || secret == "" {
		return Token{}, false
	}
	return Token{GuestID: guestID, Secret: secret}, true
}
