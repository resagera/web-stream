package http

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	stdhttp "net/http"
	"strings"
	"time"

	"home-stream/server/internal/auth"
	"home-stream/server/internal/chat"
	"home-stream/server/internal/invite"
	"home-stream/server/internal/journal"
	"home-stream/server/internal/livekit"
	"home-stream/server/internal/livestatus"
	"home-stream/server/internal/media"
	"home-stream/server/internal/profile"
)

const guestCookieName = "guest_token"

type loginRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	PhotoURL string `json:"photo_url"`
	Role     string `json:"role"`
}

type inviteLoginRequest struct {
	Name     string `json:"name"`
	Token    string `json:"token"`
	PhotoURL string `json:"photo_url"`
}

type profileRequest struct {
	Name     string `json:"name"`
	PhotoURL string `json:"photo_url"`
}

type createInviteRequest struct {
	Role    string `json:"role"`
	Label   string `json:"label"`
	MaxUses int    `json:"max_uses"`
}

type disableInviteRequest struct {
	Token string `json:"token"`
}

type eventRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type liveKitTokenRequest struct {
	CanPublish bool `json:"can_publish"`
}

type chatPost struct {
	Text string `json:"text"`
}

func (s *Server) health(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) guestLogin(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodPost {
		methodNotAllowed(w)
		return
	}

	var req loginRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, stdhttp.StatusBadRequest, "invalid_json")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, stdhttp.StatusBadRequest, "name_required")
		return
	}
	photoURL, ok := s.savePhoto(w, req.PhotoURL)
	if !ok {
		return
	}
	role := profile.NormalizeRole(strings.TrimSpace(req.Role))
	if !s.passwordMatchesRole(role, req.Password) {
		writeError(w, stdhttp.StatusUnauthorized, "bad_password")
		return
	}

	guest, err := s.guests.Create(role, req.Name, photoURL, clientIP(r))
	if err != nil {
		writeError(w, stdhttp.StatusInternalServerError, "guest_create_failed")
		return
	}

	token := auth.Issue(guest)
	s.setGuestCookie(w, token)
	s.recordGuestEvent(r, "guest_login", guest, "password")

	writeJSON(w, stdhttp.StatusOK, map[string]any{
		"guest": guest,
		"token": token,
	})
}

func (s *Server) guestInviteLogin(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodPost {
		methodNotAllowed(w)
		return
	}

	var req inviteLoginRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, stdhttp.StatusBadRequest, "invalid_json")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, stdhttp.StatusBadRequest, "name_required")
		return
	}
	req.Token = strings.TrimSpace(req.Token)
	if req.Token == "" {
		writeError(w, stdhttp.StatusBadRequest, "invite_required")
		return
	}
	photoURL, ok := s.savePhoto(w, req.PhotoURL)
	if !ok {
		return
	}

	inv, err := s.invites.Use(req.Token)
	if errors.Is(err, invite.ErrNotFound) || errors.Is(err, invite.ErrInactive) {
		writeError(w, stdhttp.StatusUnauthorized, "bad_invite")
		return
	}
	if err != nil {
		writeError(w, stdhttp.StatusInternalServerError, "invite_use_failed")
		return
	}

	guest, err := s.guests.Create(inv.Role, req.Name, photoURL, clientIP(r))
	if err != nil {
		writeError(w, stdhttp.StatusInternalServerError, "guest_create_failed")
		return
	}

	token := auth.Issue(guest)
	s.setGuestCookie(w, token)
	s.recordGuestEvent(r, "guest_login", guest, "invite:"+inv.Token)

	writeJSON(w, stdhttp.StatusOK, map[string]any{
		"guest":  guest,
		"invite": inv,
		"token":  token,
	})
}

func (s *Server) guestProfile(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodPost {
		methodNotAllowed(w)
		return
	}

	guest, ok := s.authenticateRequest(r)
	if !ok {
		writeError(w, stdhttp.StatusUnauthorized, "unauthorized")
		return
	}

	var req profileRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, stdhttp.StatusBadRequest, "invalid_json")
		return
	}
	photoURL, ok := s.savePhoto(w, req.PhotoURL)
	if !ok {
		return
	}

	updated, err := s.guests.Update(guest.ID, strings.TrimSpace(req.Name), photoURL)
	if errors.Is(err, profile.ErrNotFound) {
		writeError(w, stdhttp.StatusUnauthorized, "unauthorized")
		return
	}
	if err != nil {
		writeError(w, stdhttp.StatusInternalServerError, "profile_update_failed")
		return
	}
	s.recordGuestEvent(r, "profile_update", updated, "")

	writeJSON(w, stdhttp.StatusOK, map[string]any{"guest": updated})
}

func (s *Server) adminInvites(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if _, ok := s.authenticateAdmin(r); !ok {
		writeError(w, stdhttp.StatusUnauthorized, "admin_required")
		return
	}

	switch r.Method {
	case stdhttp.MethodGet:
		writeJSON(w, stdhttp.StatusOK, map[string]any{"invites": s.invites.List()})
	case stdhttp.MethodPost:
		var req createInviteRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, stdhttp.StatusBadRequest, "invalid_json")
			return
		}
		inv, err := s.invites.Create(profile.NormalizeRole(strings.TrimSpace(req.Role)), strings.TrimSpace(req.Label), req.MaxUses)
		if err != nil {
			writeError(w, stdhttp.StatusInternalServerError, "invite_create_failed")
			return
		}
		writeJSON(w, stdhttp.StatusOK, map[string]any{"invite": inv})
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) adminInviteDisable(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodPost {
		methodNotAllowed(w)
		return
	}
	if _, ok := s.authenticateAdmin(r); !ok {
		writeError(w, stdhttp.StatusUnauthorized, "admin_required")
		return
	}

	var req disableInviteRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, stdhttp.StatusBadRequest, "invalid_json")
		return
	}
	token := strings.TrimSpace(req.Token)
	if token == "" {
		writeError(w, stdhttp.StatusBadRequest, "invite_required")
		return
	}

	inv, err := s.invites.Disable(token)
	if errors.Is(err, invite.ErrNotFound) {
		writeError(w, stdhttp.StatusNotFound, "invite_not_found")
		return
	}
	if err != nil {
		writeError(w, stdhttp.StatusInternalServerError, "invite_disable_failed")
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]any{"invite": inv})
}

func (s *Server) adminEvent(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if _, ok := s.authenticateAdmin(r); !ok {
		writeError(w, stdhttp.StatusUnauthorized, "admin_required")
		return
	}

	switch r.Method {
	case stdhttp.MethodGet:
		writeJSON(w, stdhttp.StatusOK, map[string]any{"event": s.event.Get()})
	case stdhttp.MethodPost:
		var req eventRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, stdhttp.StatusBadRequest, "invalid_json")
			return
		}
		updated, err := s.event.Update(strings.TrimSpace(req.Title), strings.TrimSpace(req.Description))
		if err != nil {
			writeError(w, stdhttp.StatusInternalServerError, "event_update_failed")
			return
		}
		writeJSON(w, stdhttp.StatusOK, map[string]any{"event": updated})
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) adminStatus(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodGet {
		methodNotAllowed(w)
		return
	}
	if _, ok := s.authenticateAdmin(r); !ok {
		writeError(w, stdhttp.StatusUnauthorized, "admin_required")
		return
	}

	online := s.hub.Presence()
	liveCameras := s.live.Cameras()
	viewers := make([]chat.Presence, 0, len(online))
	cameras := make([]chat.Presence, 0, len(online))
	for _, item := range online {
		if profile.CanPublish(item.Role) {
			cameras = append(cameras, item)
			continue
		}
		viewers = append(viewers, item)
	}

	writeJSON(w, stdhttp.StatusOK, map[string]any{
		"online":            online,
		"viewers":           viewers,
		"cameras":           cameras,
		"livekit_cameras":   liveCameras,
		"online_count":      len(online),
		"viewer_count":      len(viewers),
		"camera_count":      len(liveCameras),
		"chat_camera_count": len(cameras),
		"message_count":     len(s.hub.History()),
	})
}

func (s *Server) adminJournal(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodGet {
		methodNotAllowed(w)
		return
	}
	if _, ok := s.authenticateAdmin(r); !ok {
		writeError(w, stdhttp.StatusUnauthorized, "admin_required")
		return
	}

	writeJSON(w, stdhttp.StatusOK, map[string]any{
		"entries": s.journal.Last(80),
	})
}

func (s *Server) livekitToken(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodPost {
		methodNotAllowed(w)
		return
	}

	guest, ok := s.authenticateRequest(r)
	if !ok {
		writeError(w, stdhttp.StatusUnauthorized, "unauthorized")
		return
	}

	var req liveKitTokenRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, stdhttp.StatusBadRequest, "invalid_json")
		return
	}
	_ = req

	token, err := livekit.Issue(livekit.Config{
		APIKey:    s.cfg.LiveKitAPIKey,
		APISecret: s.cfg.LiveKitAPISecret,
		Room:      s.cfg.LiveKitRoom,
	}, guest.ID, guest.Name, profile.CanPublish(guest.Role), 6*time.Hour)
	if err != nil {
		writeError(w, stdhttp.StatusServiceUnavailable, "livekit_not_configured")
		return
	}

	writeJSON(w, stdhttp.StatusOK, map[string]string{
		"token": token,
		"url":   s.cfg.LiveKitURL,
		"room":  s.cfg.LiveKitRoom,
	})
}

func (s *Server) livekitWebhook(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodPost {
		methodNotAllowed(w)
		return
	}
	if err := livekit.ValidateWebhookAuth(s.cfg.LiveKitAPIKey, s.cfg.LiveKitAPISecret, r.Header.Get("Authorization")); err != nil {
		writeError(w, stdhttp.StatusUnauthorized, "bad_livekit_webhook_auth")
		return
	}

	body, err := io.ReadAll(stdhttp.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeError(w, stdhttp.StatusBadRequest, "webhook_too_large")
		return
	}
	defer r.Body.Close()

	var event livekit.WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		writeError(w, stdhttp.StatusBadRequest, "invalid_json")
		return
	}
	s.applyLiveKitWebhook(event)
	s.recordLiveKitEvent(event)
	writeJSON(w, stdhttp.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) applyLiveKitWebhook(event livekit.WebhookEvent) {
	participantSID := ""
	identity := ""
	if event.Participant != nil {
		participantSID = event.Participant.SID
		identity = event.Participant.Identity
	}

	switch event.Event {
	case "participant_joined":
		if event.Participant != nil {
			s.live.ParticipantJoined(event.Participant.SID, event.Participant.Identity, event.Participant.Name)
		}
	case "participant_left":
		s.live.ParticipantLeft(participantSID, identity)
	case "track_published":
		if event.Track != nil {
			s.live.TrackPublished(participantSID, identity, livestatus.Track{
				SID:    event.Track.SID,
				Type:   strings.ToLower(event.Track.Type),
				Source: strings.ToLower(event.Track.Source),
				Name:   event.Track.Name,
			})
		}
	case "track_unpublished":
		if event.Track != nil {
			s.live.TrackUnpublished(participantSID, identity, event.Track.SID)
		}
	}
}

func (s *Server) chatWS(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodGet {
		methodNotAllowed(w)
		return
	}

	guest, ok := s.authenticateRequest(r)
	if !ok {
		writeError(w, stdhttp.StatusUnauthorized, "unauthorized")
		return
	}

	client, err := chat.Upgrade(w, r)
	if err != nil {
		return
	}
	if err := s.hub.Add(client, guest); err != nil {
		_ = client.Close()
		return
	}
	s.recordGuestEvent(r, "chat_connect", guest, "")
	defer func() {
		s.hub.Remove(client)
		s.recordGuestEvent(r, "chat_disconnect", guest, "")
	}()

	for {
		var req chatPost
		if err := client.ReadJSON(&req); err != nil {
			return
		}
		text := strings.TrimSpace(req.Text)
		if text == "" {
			_ = client.Send(chat.Envelope{Type: "error", Error: "empty_message"})
			continue
		}
		if len([]rune(text)) > 1000 {
			_ = client.Send(chat.Envelope{Type: "error", Error: "message_too_long"})
			continue
		}
		if _, err := s.hub.Post(guest, text); err != nil {
			_ = client.Send(chat.Envelope{Type: "error", Error: "message_save_failed"})
		}
	}
}

func (s *Server) authenticateRequest(r *stdhttp.Request) (profile.Guest, bool) {
	raw := ""
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		raw = strings.TrimPrefix(authHeader, "Bearer ")
	}
	if raw == "" {
		raw = r.URL.Query().Get("token")
	}
	if raw == "" {
		cookie, err := r.Cookie(guestCookieName)
		if err == nil {
			raw = cookie.Value
		}
	}

	parsed, ok := auth.Parse(raw)
	if !ok {
		return profile.Guest{}, false
	}
	return s.guests.Authenticate(parsed.GuestID, parsed.Secret)
}

func (s *Server) authenticateAdmin(r *stdhttp.Request) (profile.Guest, bool) {
	guest, ok := s.authenticateRequest(r)
	if !ok || profile.NormalizeRole(guest.Role) != profile.RoleAdmin {
		return profile.Guest{}, false
	}
	return guest, true
}

func (s *Server) passwordMatchesRole(role, password string) bool {
	switch profile.NormalizeRole(role) {
	case profile.RoleGuest:
		return password == s.cfg.GuestPassword
	case profile.RoleBroadcaster:
		return s.cfg.BroadcasterPassword != "" && password == s.cfg.BroadcasterPassword
	case profile.RoleAdmin:
		return s.cfg.AdminPassword != "" && password == s.cfg.AdminPassword
	default:
		return false
	}
}

func (s *Server) setGuestCookie(w stdhttp.ResponseWriter, token string) {
	stdhttp.SetCookie(w, &stdhttp.Cookie{
		Name:     guestCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.SecureCookies,
		SameSite: stdhttp.SameSiteLaxMode,
		MaxAge:   int((30 * 24 * time.Hour).Seconds()),
	})
}

func (s *Server) savePhoto(w stdhttp.ResponseWriter, raw string) (string, bool) {
	photoURL, err := s.media.SavePhoto(raw)
	if err == nil {
		return photoURL, true
	}
	switch {
	case errors.Is(err, media.ErrTooLarge):
		writeError(w, stdhttp.StatusBadRequest, "photo_too_large")
	case errors.Is(err, media.ErrInvalidImage):
		writeError(w, stdhttp.StatusBadRequest, "photo_invalid")
	default:
		writeError(w, stdhttp.StatusInternalServerError, "photo_save_failed")
	}
	return "", false
}

func (s *Server) recordGuestEvent(r *stdhttp.Request, eventType string, guest profile.Guest, detail string) {
	_, _ = s.journal.Record(journal.Event{
		Type:    eventType,
		GuestID: guest.ID,
		Name:    guest.Name,
		Role:    profile.NormalizeRole(guest.Role),
		IP:      clientIP(r),
		Detail:  detail,
	})
}

func (s *Server) recordLiveKitEvent(event livekit.WebhookEvent) {
	journalEvent := journal.Event{
		Type:   "livekit_" + event.Event,
		Detail: event.Event,
	}
	if event.Participant != nil {
		journalEvent.GuestID = event.Participant.Identity
		journalEvent.Name = event.Participant.Name
		journalEvent.Metadata = map[string]any{
			"participant_sid": event.Participant.SID,
		}
	}
	if event.Track != nil {
		if journalEvent.Metadata == nil {
			journalEvent.Metadata = make(map[string]any)
		}
		journalEvent.Metadata["track_sid"] = event.Track.SID
		journalEvent.Metadata["track_type"] = event.Track.Type
		journalEvent.Metadata["track_source"] = event.Track.Source
	}
	_, _ = s.journal.Record(journalEvent)
}

func clientIP(r *stdhttp.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
