package livekit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var ErrBadWebhookAuth = errors.New("bad livekit webhook auth")

type WebhookEvent struct {
	Event       string              `json:"event"`
	ID          string              `json:"id,omitempty"`
	CreatedAt   int64               `json:"createdAt,omitempty"`
	Room        *WebhookRoom        `json:"room,omitempty"`
	Participant *WebhookParticipant `json:"participant,omitempty"`
	Track       *WebhookTrack       `json:"track,omitempty"`
}

type WebhookRoom struct {
	SID  string `json:"sid,omitempty"`
	Name string `json:"name,omitempty"`
}

type WebhookParticipant struct {
	SID      string `json:"sid,omitempty"`
	Identity string `json:"identity,omitempty"`
	Name     string `json:"name,omitempty"`
}

type WebhookTrack struct {
	SID    string `json:"sid,omitempty"`
	Type   string `json:"type,omitempty"`
	Source string `json:"source,omitempty"`
	Name   string `json:"name,omitempty"`
}

func ValidateWebhookAuth(apiKey, apiSecret, header string) error {
	if apiKey == "" || apiSecret == "" {
		return ErrBadWebhookAuth
	}
	token := strings.TrimSpace(header)
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ErrBadWebhookAuth
	}

	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(apiSecret))
	mac.Write([]byte(unsigned))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return ErrBadWebhookAuth
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ErrBadWebhookAuth
	}
	var claims struct {
		Issuer    string `json:"iss"`
		NotBefore int64  `json:"nbf"`
		ExpiresAt int64  `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ErrBadWebhookAuth
	}
	now := time.Now().Unix()
	if claims.Issuer != apiKey {
		return ErrBadWebhookAuth
	}
	if claims.NotBefore != 0 && now < claims.NotBefore {
		return ErrBadWebhookAuth
	}
	if claims.ExpiresAt != 0 && now > claims.ExpiresAt {
		return ErrBadWebhookAuth
	}
	return nil
}
