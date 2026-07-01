package http

import (
	"net/http"

	"home-stream/server/internal/chat"
	"home-stream/server/internal/config"
	"home-stream/server/internal/event"
	"home-stream/server/internal/invite"
	"home-stream/server/internal/journal"
	"home-stream/server/internal/livestatus"
	"home-stream/server/internal/media"
	"home-stream/server/internal/profile"
)

type Server struct {
	cfg     config.Config
	mux     *http.ServeMux
	guests  *profile.Store
	invites *invite.Store
	event   *event.Store
	journal *journal.Store
	media   *media.Store
	live    *livestatus.Store
	hub     *chat.Hub
}

func NewServer(cfg config.Config) (*Server, error) {
	guests, err := profile.NewStore(cfg.GuestsPath())
	if err != nil {
		return nil, err
	}
	invites, err := invite.NewStore(cfg.InvitesPath())
	if err != nil {
		return nil, err
	}
	eventStore, err := event.NewStore(cfg.EventPath())
	if err != nil {
		return nil, err
	}
	journalStore, err := journal.NewStore(cfg.JournalPath(), 200)
	if err != nil {
		return nil, err
	}
	mediaStore, err := media.NewStore(cfg.PhotosDir(), "/media/photos", cfg.MaxPhotoURLBytes)
	if err != nil {
		return nil, err
	}
	hub, err := chat.NewHub(1000, chat.NewStorage(cfg.ChatPath()))
	if err != nil {
		return nil, err
	}

	s := &Server{
		cfg:     cfg,
		mux:     http.NewServeMux(),
		guests:  guests,
		invites: invites,
		event:   eventStore,
		journal: journalStore,
		media:   mediaStore,
		live:    livestatus.NewStore(),
		hub:     hub,
	}
	s.routes()
	return s, nil
}

func (s *Server) Handler() http.Handler {
	return s.withCORS(s.mux)
}
