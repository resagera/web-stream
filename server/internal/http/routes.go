package http

import (
	"net/http"
	"strings"

	"home-stream/server/internal/web"
)

func (s *Server) routes() {
	s.mux.HandleFunc("/health", s.health)
	s.mux.HandleFunc("/api/guest/login", s.guestLogin)
	s.mux.HandleFunc("/api/guest/invite-login", s.guestInviteLogin)
	s.mux.HandleFunc("/api/guest/profile", s.guestProfile)
	s.mux.HandleFunc("/api/livekit/token", s.livekitToken)
	s.mux.HandleFunc("/api/livekit/webhook", s.livekitWebhook)
	s.mux.HandleFunc("/api/admin/event", s.adminEvent)
	s.mux.HandleFunc("/api/admin/status", s.adminStatus)
	s.mux.HandleFunc("/api/admin/journal", s.adminJournal)
	s.mux.HandleFunc("/api/admin/invites", s.adminInvites)
	s.mux.HandleFunc("/api/admin/invites/disable", s.adminInviteDisable)
	s.mux.HandleFunc("/ws/chat", s.chatWS)
	s.mux.HandleFunc("/media/photos/", s.photoMedia)
	s.mux.Handle("/", web.Handler())
}

func (s *Server) photoMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/media/photos/")
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, s.cfg.PhotosDir()+"/"+name)
}
