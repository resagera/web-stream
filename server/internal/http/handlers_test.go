package http

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"home-stream/server/internal/config"
	"home-stream/server/internal/livekit"
)

func TestLiveKitPublishGrantComesFromRole(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	guestToken := loginForTest(t, server, `{"name":"Guest","password":"guest-pass","role":"guest"}`)
	guestJWT := liveKitTokenForTest(t, server, guestToken, `{"can_publish":true}`)
	if canPublish := jwtCanPublish(t, guestJWT); canPublish {
		t.Fatalf("guest canPublish = true, want false")
	}

	broadcasterToken := loginForTest(t, server, `{"name":"Camera","password":"broadcaster-pass","role":"broadcaster"}`)
	broadcasterJWT := liveKitTokenForTest(t, server, broadcasterToken, `{"can_publish":false}`)
	if canPublish := jwtCanPublish(t, broadcasterJWT); !canPublish {
		t.Fatalf("broadcaster canPublish = false, want true")
	}
}

func TestInviteLoginConsumesInviteAndAppliesRole(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	adminToken := loginForTest(t, server, `{"name":"Admin","password":"admin-pass","role":"admin"}`)
	inviteToken := createInviteForTest(t, server, adminToken, `{"role":"broadcaster","label":"Kitchen camera","max_uses":1}`)

	broadcasterToken := inviteLoginForTest(t, server, `{"name":"Kitchen","token":"`+inviteToken+`"}`)
	broadcasterJWT := liveKitTokenForTest(t, server, broadcasterToken, `{"can_publish":false}`)
	if canPublish := jwtCanPublish(t, broadcasterJWT); !canPublish {
		t.Fatalf("invite broadcaster canPublish = false, want true")
	}

	req := httptest.NewRequest("POST", "/api/guest/invite-login", strings.NewReader(`{"name":"Second","token":"`+inviteToken+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("second invite login status = %d, want 401; body = %s", rec.Code, rec.Body.String())
	}
}

func TestStaticFrontendServed(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	for _, path := range []string{"/", "/app.css", "/app.js", "/admin.html", "/admin.js"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200", path, rec.Code)
		}
		if rec.Body.Len() == 0 {
			t.Fatalf("GET %s returned empty body", path)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("GET /admin status = %d, want 307", rec.Code)
	}
	if location := rec.Header().Get("Location"); location != "/admin.html" {
		t.Fatalf("GET /admin Location = %q, want /admin.html", location)
	}
}

func TestPhotoMediaDoesNotListDirectory(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/media/photos/", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /media/photos/ status = %d, want 404", rec.Code)
	}
}

func TestServerRecoversCorruptDataFiles(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	mustWriteFile(t, filepath.Join(dataDir, "guests.json"), []byte(`{"broken"`))
	mustWriteFile(t, filepath.Join(dataDir, "passwords.json"), []byte(`{"broken"`))
	mustWriteFile(t, filepath.Join(dataDir, "invites.json"), []byte(`{"broken"`))
	mustWriteFile(t, filepath.Join(dataDir, "event.json"), []byte(`{"broken"`))
	mustWriteFile(t, filepath.Join(dataDir, "chat.jsonl"), []byte(strings.Repeat("x", 70*1024)))
	mustWriteFile(t, filepath.Join(dataDir, "journal.jsonl"), []byte(strings.Repeat("x", 2*1024*1024)))

	server, err := NewServer(config.Config{
		Addr:                ":0",
		DataDir:             dataDir,
		MaxPhotoURLBytes:    350000,
		GuestPassword:       "guest-pass",
		BroadcasterPassword: "broadcaster-pass",
		AdminPassword:       "admin-pass",
		LiveKitURL:          "ws://livekit.test",
		LiveKitAPIKey:       "devkey",
		LiveKitAPISecret:    "secret",
		LiveKitRoom:         "family-event",
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	for _, name := range []string{"guests.json", "passwords.json", "invites.json", "event.json", "chat.jsonl", "journal.jsonl"} {
		matches, err := filepath.Glob(filepath.Join(dataDir, name+".corrupt.*"))
		if err != nil {
			t.Fatalf("glob corrupt backup for %s: %v", name, err)
		}
		if len(matches) != 1 {
			t.Fatalf("corrupt backups for %s = %d, want 1", name, len(matches))
		}
	}

	token := loginForTest(t, server, `{"name":"Guest","password":"guest-pass","role":"guest"}`)
	if token == "" {
		t.Fatal("empty token after corrupt recovery")
	}
}

func TestAdminEventAndStatus(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	guestToken := loginForTest(t, server, `{"name":"Guest","password":"guest-pass","role":"guest"}`)
	adminToken := loginForTest(t, server, `{"name":"Admin","password":"admin-pass","role":"admin"}`)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/status", nil)
	req.Header.Set("Authorization", "Bearer "+guestToken)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("guest admin status = %d, want 401", rec.Code)
	}

	eventResponse := adminPostForTest(t, server, adminToken, "/api/admin/event", `{"title":"Wedding","description":"Family evening"}`)
	var eventPayload struct {
		Event struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"event"`
	}
	if err := json.Unmarshal(eventResponse, &eventPayload); err != nil {
		t.Fatalf("decode event response: %v", err)
	}
	if eventPayload.Event.Title != "Wedding" || eventPayload.Event.Description != "Family evening" {
		t.Fatalf("event payload = %+v", eventPayload.Event)
	}

	statusResponse := adminGetForTest(t, server, adminToken, "/api/admin/status")
	var statusPayload struct {
		OnlineCount int `json:"online_count"`
		ViewerCount int `json:"viewer_count"`
		CameraCount int `json:"camera_count"`
	}
	if err := json.Unmarshal(statusResponse, &statusPayload); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if statusPayload.OnlineCount != 0 || statusPayload.ViewerCount != 0 || statusPayload.CameraCount != 0 {
		t.Fatalf("status payload = %+v, want all zero without websocket clients", statusPayload)
	}
}

func TestAdminPasswordsUpdateLoginPassword(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	guestToken := loginForTest(t, server, `{"name":"Guest","password":"guest-pass","role":"guest"}`)
	adminToken := loginForTest(t, server, `{"name":"Admin","password":"admin-pass","role":"admin"}`)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/passwords", nil)
	req.Header.Set("Authorization", "Bearer "+guestToken)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("guest admin passwords = %d, want 401", rec.Code)
	}

	shortReq := httptest.NewRequest(http.MethodPost, "/api/admin/passwords", strings.NewReader(`{"guest_password":"123"}`))
	shortReq.Header.Set("Content-Type", "application/json")
	shortReq.Header.Set("Authorization", "Bearer "+adminToken)
	shortRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(shortRec, shortReq)
	if shortRec.Code != http.StatusBadRequest {
		t.Fatalf("short password status = %d, want 400", shortRec.Code)
	}

	response := adminPostForTest(t, server, adminToken, "/api/admin/passwords", `{"guest_password":"new-guest-pass"}`)
	var payload struct {
		Passwords struct {
			GuestConfigured       bool `json:"guest_configured"`
			BroadcasterConfigured bool `json:"broadcaster_configured"`
			AdminConfigured       bool `json:"admin_configured"`
		} `json:"passwords"`
	}
	if err := json.Unmarshal(response, &payload); err != nil {
		t.Fatalf("decode passwords response: %v", err)
	}
	if !payload.Passwords.GuestConfigured || !payload.Passwords.BroadcasterConfigured || !payload.Passwords.AdminConfigured {
		t.Fatalf("password status = %+v, want all configured", payload.Passwords)
	}

	oldReq := httptest.NewRequest(http.MethodPost, "/api/guest/login", strings.NewReader(`{"name":"Old","password":"guest-pass","role":"guest"}`))
	oldReq.Header.Set("Content-Type", "application/json")
	oldRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(oldRec, oldReq)
	if oldRec.Code != http.StatusUnauthorized {
		t.Fatalf("old guest password status = %d, want 401", oldRec.Code)
	}

	newToken := loginForTest(t, server, `{"name":"New","password":"new-guest-pass","role":"guest"}`)
	if newToken == "" {
		t.Fatal("new guest token is empty")
	}
}

func TestAdminJournalRecordsLogin(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	guestToken := loginForTest(t, server, `{"name":"Guest","password":"guest-pass","role":"guest"}`)
	adminToken := loginForTest(t, server, `{"name":"Admin","password":"admin-pass","role":"admin"}`)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/journal", nil)
	req.Header.Set("Authorization", "Bearer "+guestToken)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("guest admin journal = %d, want 401", rec.Code)
	}

	journalResponse := adminGetForTest(t, server, adminToken, "/api/admin/journal")
	var payload struct {
		Entries []struct {
			Type string `json:"type"`
			Name string `json:"name"`
			Role string `json:"role"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(journalResponse, &payload); err != nil {
		t.Fatalf("decode journal response: %v", err)
	}
	if len(payload.Entries) < 2 {
		t.Fatalf("journal entries = %d, want at least 2", len(payload.Entries))
	}
	if payload.Entries[0].Type != "guest_login" || payload.Entries[0].Name != "Guest" || payload.Entries[0].Role != "guest" {
		t.Fatalf("first journal entry = %+v", payload.Entries[0])
	}
}

func TestLiveKitWebhookUpdatesCameraStatus(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	adminToken := loginForTest(t, server, `{"name":"Admin","password":"admin-pass","role":"admin"}`)
	webhookToken, err := livekit.Issue(livekit.Config{
		APIKey:    "devkey",
		APISecret: "secret",
		Room:      "family-event",
	}, "livekit-webhook", "", false, time.Hour)
	if err != nil {
		t.Fatalf("issue webhook token: %v", err)
	}

	body := `{"event":"track_published","participant":{"sid":"PA_1","identity":"kitchen","name":"Kitchen"},"track":{"sid":"TR_1","type":"video","source":"camera","name":"front"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/livekit/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+webhookToken)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("webhook status = %d, body = %s", rec.Code, rec.Body.String())
	}

	statusResponse := adminGetForTest(t, server, adminToken, "/api/admin/status")
	var statusPayload struct {
		CameraCount    int `json:"camera_count"`
		LiveKitCameras []struct {
			Identity string `json:"identity"`
		} `json:"livekit_cameras"`
	}
	if err := json.Unmarshal(statusResponse, &statusPayload); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if statusPayload.CameraCount != 1 || len(statusPayload.LiveKitCameras) != 1 || statusPayload.LiveKitCameras[0].Identity != "kitchen" {
		t.Fatalf("status payload = %+v", statusPayload)
	}
}

func TestCORSWhitelistSecureCookieAndPhotoLimit(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Addr:             ":0",
		DataDir:          t.TempDir(),
		PublicOrigin:     "https://stream.example.com",
		SecureCookies:    true,
		MaxPhotoURLBytes: 12,
		GuestPassword:    "guest-pass",
		LiveKitAPIKey:    "devkey",
		LiveKitAPISecret: "secret",
		LiveKitRoom:      "family-event",
	}
	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	preflight := httptest.NewRequest(http.MethodOptions, "/api/guest/login", nil)
	preflight.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, preflight)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("bad origin preflight status = %d, want 403", rec.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/guest/login", strings.NewReader(`{"name":"Guest","password":"guest-pass","photo_url":"data:image/png;base64,MDEyMzQ1Njc4OTAxMjM0NTY3ODk="}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://stream.example.com")
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("large photo status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/guest/login", strings.NewReader(`{"name":"Guest","password":"guest-pass","photo_url":"data:image/png;base64,aGk="}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://stream.example.com")
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid photo signature status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/guest/login", strings.NewReader(`{"name":"Guest","password":"guest-pass","photo_url":"data:image/png;base64,iVBORw0KGgo="}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://stream.example.com")
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://stream.example.com" {
		t.Fatalf("allow origin = %q", got)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Secure {
		t.Fatalf("secure cookie not set: %+v", cookies)
	}
	var payload struct {
		Guest struct {
			PhotoURL string `json:"photo_url"`
		} `json:"guest"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if !strings.HasPrefix(payload.Guest.PhotoURL, "/media/photos/") {
		t.Fatalf("photo url = %q, want media url", payload.Guest.PhotoURL)
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()

	cfg := config.Config{
		Addr:                ":0",
		DataDir:             t.TempDir(),
		MaxPhotoURLBytes:    350000,
		GuestPassword:       "guest-pass",
		BroadcasterPassword: "broadcaster-pass",
		AdminPassword:       "admin-pass",
		LiveKitURL:          "ws://livekit.test",
		LiveKitAPIKey:       "devkey",
		LiveKitAPISecret:    "secret",
		LiveKitRoom:         "family-event",
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	return server
}

func loginForTest(t *testing.T, server *Server, body string) string {
	t.Helper()

	req := httptest.NewRequest("POST", "/api/guest/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("login status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var response struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if response.Token == "" {
		t.Fatalf("login token is empty")
	}
	return response.Token
}

func createInviteForTest(t *testing.T, server *Server, adminToken, body string) string {
	t.Helper()

	req := httptest.NewRequest("POST", "/api/admin/invites", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("create invite status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var response struct {
		Invite struct {
			Token string `json:"token"`
		} `json:"invite"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode create invite response: %v", err)
	}
	if response.Invite.Token == "" {
		t.Fatalf("invite token is empty")
	}
	return response.Invite.Token
}

func inviteLoginForTest(t *testing.T, server *Server, body string) string {
	t.Helper()

	req := httptest.NewRequest("POST", "/api/guest/invite-login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("invite login status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var response struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode invite login response: %v", err)
	}
	if response.Token == "" {
		t.Fatalf("invite login token is empty")
	}
	return response.Token
}

func adminPostForTest(t *testing.T, server *Server, adminToken, path, body string) []byte {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST %s status = %d, body = %s", path, rec.Code, rec.Body.String())
	}
	return rec.Body.Bytes()
}

func adminGetForTest(t *testing.T, server *Server, adminToken, path string) []byte {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, body = %s", path, rec.Code, rec.Body.String())
	}
	return rec.Body.Bytes()
}

func liveKitTokenForTest(t *testing.T, server *Server, guestToken, body string) string {
	t.Helper()

	req := httptest.NewRequest("POST", "/api/livekit/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+guestToken)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("livekit token status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var response struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode livekit response: %v", err)
	}
	if response.Token == "" {
		t.Fatalf("livekit token is empty")
	}
	return response.Token
}

func jwtCanPublish(t *testing.T, token string) bool {
	t.Helper()

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("JWT has %d parts, want 3", len(parts))
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode JWT payload: %v", err)
	}

	var claims struct {
		Video struct {
			CanPublish bool `json:"canPublish"`
		} `json:"video"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("decode JWT claims: %v", err)
	}
	return claims.Video.CanPublish
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
