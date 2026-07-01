package config

import (
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Addr                string
	DataDir             string
	PublicOrigin        string
	SecureCookies       bool
	MaxPhotoURLBytes    int
	GuestPassword       string
	BroadcasterPassword string
	AdminPassword       string

	LiveKitURL       string
	LiveKitAPIKey    string
	LiveKitAPISecret string
	LiveKitRoom      string
}

func Load() Config {
	dataDir := getEnv("DATA_DIR", "data")

	return Config{
		Addr:                getEnv("ADDR", ":8080"),
		DataDir:             dataDir,
		PublicOrigin:        os.Getenv("PUBLIC_ORIGIN"),
		SecureCookies:       getBoolEnv("SECURE_COOKIES", false),
		MaxPhotoURLBytes:    getIntEnv("MAX_PHOTO_URL_BYTES", 350000),
		GuestPassword:       getEnv("GUEST_PASSWORD", "change-me"),
		BroadcasterPassword: os.Getenv("BROADCASTER_PASSWORD"),
		AdminPassword:       os.Getenv("ADMIN_PASSWORD"),

		LiveKitURL:       os.Getenv("LIVEKIT_URL"),
		LiveKitAPIKey:    os.Getenv("LIVEKIT_API_KEY"),
		LiveKitAPISecret: os.Getenv("LIVEKIT_API_SECRET"),
		LiveKitRoom:      getEnv("LIVEKIT_ROOM", "family-event"),
	}
}

func (c Config) GuestsPath() string {
	return filepath.Join(c.DataDir, "guests.json")
}

func (c Config) ChatPath() string {
	return filepath.Join(c.DataDir, "chat.jsonl")
}

func (c Config) InvitesPath() string {
	return filepath.Join(c.DataDir, "invites.json")
}

func (c Config) EventPath() string {
	return filepath.Join(c.DataDir, "event.json")
}

func (c Config) JournalPath() string {
	return filepath.Join(c.DataDir, "journal.jsonl")
}

func (c Config) PhotosDir() string {
	return filepath.Join(c.DataDir, "photos")
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getIntEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
