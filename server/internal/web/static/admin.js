const state = {
  token: localStorage.getItem("guest_token") || "",
  guest: readJSON(localStorage.getItem("guest"))
};

const els = {
  statusLine: document.querySelector("#adminStatusLine"),
  loginPanel: document.querySelector("#adminLoginPanel"),
  dashboard: document.querySelector("#adminDashboard"),
  loginForm: document.querySelector("#adminLoginForm"),
  nameInput: document.querySelector("#adminNameInput"),
  loginPasswordInput: document.querySelector("#adminLoginPasswordInput"),
  loginError: document.querySelector("#adminLoginError"),
  logoutButton: document.querySelector("#adminLogoutButton"),
  refreshButton: document.querySelector("#adminRefreshButton"),

  eventForm: document.querySelector("#adminEventForm"),
  eventTitleInput: document.querySelector("#adminEventTitleInput"),
  eventDescriptionInput: document.querySelector("#adminEventDescriptionInput"),

  passwordsForm: document.querySelector("#adminPasswordsForm"),
  guestPasswordInput: document.querySelector("#adminGuestPasswordInput"),
  broadcasterPasswordInput: document.querySelector("#adminBroadcasterPasswordInput"),
  adminPasswordInput: document.querySelector("#adminNewPasswordInput"),
  passwordStatus: document.querySelector("#adminPasswordStatus"),

  onlineCount: document.querySelector("#adminOnlineCount"),
  viewerCount: document.querySelector("#adminViewerCount"),
  cameraCount: document.querySelector("#adminCameraCount"),
  liveCameraCount: document.querySelector("#adminLiveCameraCount"),
  chatCameraCount: document.querySelector("#adminChatCameraCount"),
  viewerPresenceCount: document.querySelector("#adminViewerPresenceCount"),
  liveCameraList: document.querySelector("#adminLiveCameraList"),
  cameraPresenceList: document.querySelector("#adminCameraPresenceList"),
  viewerPresenceList: document.querySelector("#adminViewerPresenceList"),

  inviteForm: document.querySelector("#adminInviteForm"),
  inviteLabelInput: document.querySelector("#adminInviteLabelInput"),
  inviteRoleInput: document.querySelector("#adminInviteRoleInput"),
  inviteUsesInput: document.querySelector("#adminInviteUsesInput"),
  inviteList: document.querySelector("#adminInviteList"),
  journalList: document.querySelector("#adminJournalList")
};

boot();

function boot() {
  els.loginForm.addEventListener("submit", onLogin);
  els.logoutButton.addEventListener("click", logout);
  els.refreshButton.addEventListener("click", loadAdmin);
  els.eventForm.addEventListener("submit", onEventSave);
  els.passwordsForm.addEventListener("submit", onPasswordSave);
  els.inviteForm.addEventListener("submit", onInviteCreate);
  renderSession();
}

function renderSession() {
  const admin = state.token && state.guest?.role === "admin";
  els.loginPanel.hidden = admin;
  els.dashboard.hidden = !admin;
  els.logoutButton.hidden = !state.token;
  els.statusLine.textContent = admin ? `Админ: ${state.guest.name}` : "Не авторизован";
  if (admin) {
    loadAdmin().catch(showError);
  }
}

async function onLogin(event) {
  event.preventDefault();
  els.loginError.textContent = "";
  try {
    const payload = await post("/api/guest/login", {
      name: els.nameInput.value.trim(),
      password: els.loginPasswordInput.value,
      role: "admin"
    }, false);
    state.token = payload.token;
    state.guest = payload.guest;
    localStorage.setItem("guest_token", payload.token);
    localStorage.setItem("guest", JSON.stringify(payload.guest));
    els.loginPasswordInput.value = "";
    renderSession();
  } catch (err) {
    els.loginError.textContent = err.message;
  }
}

async function loadAdmin() {
  if (!state.token || state.guest?.role !== "admin") return;
  await Promise.all([loadEvent(), loadPasswords(), loadStatus(), loadInvites(), loadJournal()]);
}

async function loadEvent() {
  const payload = await get("/api/admin/event");
  els.eventTitleInput.value = payload.event?.title || "";
  els.eventDescriptionInput.value = payload.event?.description || "";
}

async function onEventSave(event) {
  event.preventDefault();
  const payload = await post("/api/admin/event", {
    title: els.eventTitleInput.value.trim(),
    description: els.eventDescriptionInput.value.trim()
  });
  els.eventTitleInput.value = payload.event?.title || "";
  els.eventDescriptionInput.value = payload.event?.description || "";
}

async function loadPasswords() {
  const payload = await get("/api/admin/passwords");
  renderPasswords(payload.passwords);
}

async function onPasswordSave(event) {
  event.preventDefault();
  const payload = {};
  const guestPassword = els.guestPasswordInput.value.trim();
  const broadcasterPassword = els.broadcasterPasswordInput.value.trim();
  const adminPassword = els.adminPasswordInput.value.trim();
  if (guestPassword) payload.guest_password = guestPassword;
  if (broadcasterPassword) payload.broadcaster_password = broadcasterPassword;
  if (adminPassword) payload.admin_password = adminPassword;
  if (!Object.keys(payload).length) {
    els.passwordStatus.textContent = "нет изменений";
    return;
  }
  const response = await post("/api/admin/passwords", payload);
  els.guestPasswordInput.value = "";
  els.broadcasterPasswordInput.value = "";
  els.adminPasswordInput.value = "";
  renderPasswords(response.passwords);
}

function renderPasswords(passwords) {
  const labels = [
    passwords?.guest_configured ? "гости: задан" : "гости: нет",
    passwords?.broadcaster_configured ? "камеры: задан" : "камеры: нет",
    passwords?.admin_configured ? "админ: задан" : "админ: нет"
  ];
  els.passwordStatus.textContent = labels.join(" · ");
}

async function loadStatus() {
  const payload = await get("/api/admin/status");
  els.onlineCount.textContent = String(payload.online_count || 0);
  els.viewerCount.textContent = String(payload.viewer_count || 0);
  els.cameraCount.textContent = String(payload.camera_count || 0);
  els.liveCameraCount.textContent = `${payload.camera_count || 0} live`;
  els.chatCameraCount.textContent = `${payload.chat_camera_count || 0} online`;
  els.viewerPresenceCount.textContent = `${payload.viewer_count || 0} online`;
  renderStatusList(els.liveCameraList, payload.livekit_cameras || [], "Нет камер в эфире", renderLiveCamera);
  renderStatusList(els.cameraPresenceList, payload.cameras || [], "Нет устройств онлайн", renderPresence);
  renderStatusList(els.viewerPresenceList, payload.viewers || [], "Нет зрителей онлайн", renderPresence);
}

async function onInviteCreate(event) {
  event.preventDefault();
  await post("/api/admin/invites", {
    role: els.inviteRoleInput.value,
    label: els.inviteLabelInput.value.trim(),
    max_uses: Number(els.inviteUsesInput.value || 0)
  });
  els.inviteLabelInput.value = "";
  await loadInvites();
}

async function loadInvites() {
  const payload = await get("/api/admin/invites");
  els.inviteList.replaceChildren();
  for (const invite of payload.invites || []) {
    renderInvite(invite);
  }
}

function renderInvite(invite) {
  const item = document.createElement("div");
  item.className = "inviteItem";
  const title = document.createElement("strong");
  title.textContent = `${invite.label || invite.role} · ${invite.active ? "active" : "off"}`;
  const token = document.createElement("div");
  token.className = "inviteToken";
  token.textContent = invite.token;
  const meta = document.createElement("div");
  meta.className = "muted";
  meta.textContent = `${invite.role} · ${invite.used_count}/${invite.max_uses || "∞"}`;
  const disable = document.createElement("button");
  disable.type = "button";
  disable.textContent = "Отключить";
  disable.disabled = !invite.active;
  disable.addEventListener("click", async () => {
    await post("/api/admin/invites/disable", { token: invite.token });
    await loadInvites();
  });
  item.append(title, token, meta, disable);
  els.inviteList.append(item);
}

async function loadJournal() {
  const payload = await get("/api/admin/journal");
  els.journalList.replaceChildren();
  for (const entry of [...(payload.entries || [])].reverse()) {
    renderJournalEntry(entry);
  }
}

function renderStatusList(container, items, emptyText, renderItem) {
  container.replaceChildren();
  if (!items.length) {
    const empty = document.createElement("div");
    empty.className = "statusEmpty";
    empty.textContent = emptyText;
    container.append(empty);
    return;
  }
  for (const item of items) {
    container.append(renderItem(item));
  }
}

function renderPresence(item) {
  const row = document.createElement("div");
  row.className = "statusItem";
  const name = document.createElement("strong");
  name.textContent = item.name || item.guest_id;
  const role = document.createElement("span");
  role.className = "pill";
  role.textContent = item.role;
  row.append(name, role);
  return row;
}

function renderLiveCamera(item) {
  const row = document.createElement("div");
  row.className = "statusItem stacked";
  const title = document.createElement("strong");
  title.textContent = item.name || item.identity || item.sid || "camera";
  const meta = document.createElement("span");
  meta.className = "muted";
  const tracks = Object.values(item.tracks || {});
  const trackNames = tracks.map((track) => track.name || track.source || track.type).filter(Boolean);
  meta.textContent = trackNames.length ? trackNames.join(", ") : item.identity || "video";
  row.append(title, meta);
  return row;
}

function renderJournalEntry(entry) {
  const row = document.createElement("div");
  row.className = "journalItem";
  const title = document.createElement("strong");
  title.textContent = journalTitle(entry);
  const meta = document.createElement("span");
  meta.className = "muted";
  meta.textContent = new Date(entry.created_at).toLocaleString([], {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit"
  });
  row.append(title, meta);
  els.journalList.append(row);
}

function journalTitle(entry) {
  const name = entry.name || entry.guest_id || "system";
  const labels = {
    guest_login: "вход",
    profile_update: "профиль",
    chat_connect: "чат online",
    chat_disconnect: "чат offline",
    livekit_participant_joined: "LiveKit вход",
    livekit_participant_left: "LiveKit выход",
    livekit_track_published: "камера live",
    livekit_track_unpublished: "камера stop"
  };
  const label = labels[entry.type] || entry.type;
  return `${label}: ${name}`;
}

async function get(path) {
  const response = await fetch(path, { headers: headers() });
  return parseResponse(response);
}

async function post(path, body, authorized = true) {
  const response = await fetch(path, {
    method: "POST",
    headers: headers(authorized),
    body: JSON.stringify(body)
  });
  return parseResponse(response);
}

function headers(authorized = true) {
  const result = { "Content-Type": "application/json" };
  if (authorized && state.token) {
    result.Authorization = `Bearer ${state.token}`;
  }
  return result;
}

async function parseResponse(response) {
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    if (response.status === 401 && state.token) {
      logout();
    }
    throw new Error(payload.error || `HTTP ${response.status}`);
  }
  return payload;
}

function logout() {
  localStorage.removeItem("guest");
  localStorage.removeItem("guest_token");
  state.token = "";
  state.guest = null;
  renderSession();
}

function showError(err) {
  els.statusLine.textContent = err.message;
}

function readJSON(raw) {
  if (!raw) return null;
  try {
    return JSON.parse(raw);
  } catch {
    return null;
  }
}
