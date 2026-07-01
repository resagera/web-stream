package livekit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

type Config struct {
	APIKey    string
	APISecret string
	Room      string
}

type Claims struct {
	Issuer    string     `json:"iss"`
	Subject   string     `json:"sub"`
	Name      string     `json:"name,omitempty"`
	NotBefore int64      `json:"nbf"`
	ExpiresAt int64      `json:"exp"`
	Video     VideoGrant `json:"video"`
}

type VideoGrant struct {
	Room         string `json:"room"`
	RoomJoin     bool   `json:"roomJoin"`
	CanPublish   bool   `json:"canPublish"`
	CanSubscribe bool   `json:"canSubscribe"`
}

func Issue(cfg Config, identity, name string, canPublish bool, ttl time.Duration) (string, error) {
	if cfg.APIKey == "" || cfg.APISecret == "" || cfg.Room == "" {
		return "", errors.New("livekit is not configured")
	}

	now := time.Now().UTC()
	claims := Claims{
		Issuer:    cfg.APIKey,
		Subject:   identity,
		Name:      name,
		NotBefore: now.Unix(),
		ExpiresAt: now.Add(ttl).Unix(),
		Video: VideoGrant{
			Room:         cfg.Room,
			RoomJoin:     true,
			CanPublish:   canPublish,
			CanSubscribe: true,
		},
	}

	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	unsigned := encode(headerJSON) + "." + encode(claimsJSON)
	mac := hmac.New(sha256.New, []byte(cfg.APISecret))
	mac.Write([]byte(unsigned))
	return unsigned + "." + encode(mac.Sum(nil)), nil
}

func encode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}
