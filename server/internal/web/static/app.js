import {
  Room,
  RoomEvent,
  Track,
  createLocalTracks
} from "https://cdn.jsdelivr.net/npm/livekit-client/+esm";

const state = {
  authMode: "password",
  token: localStorage.getItem("guest_token") || "",
  guest: readJSON(localStorage.getItem("guest")),
  room: null,
  chat: null,
  photoURL: "",
  photoStream: null,
  previewStream: null,
  localTracks: [],
  published: false
};

const els = {
  statusLine: document.querySelector("#statusLine"),
  connectButton: document.querySelector("#connectButton"),
  publishButton: document.querySelector("#publishButton"),
  logoutButton: document.querySelector("#logoutButton"),
  videoGrid: document.querySelector("#videoGrid"),

  loginPanel: document.querySelector("#loginPanel"),
  loginForm: document.querySelector("#loginForm"),
  loginError: document.querySelector("#loginError"),
  passwordTab: document.querySelector("#passwordTab"),
  inviteTab: document.querySelector("#inviteTab"),
  passwordField: document.querySelector("#passwordField"),
  inviteField: document.querySelector("#inviteField"),
  roleField: document.querySelector("#roleField"),
  nameInput: document.querySelector("#nameInput"),
  passwordInput: document.querySelector("#passwordInput"),
  inviteInput: document.querySelector("#inviteInput"),
  roleInput: document.querySelector("#roleInput"),

  profilePanel: document.querySelector("#profilePanel"),
  profileForm: document.querySelector("#profileForm"),
  profileNameInput: document.querySelector("#profileNameInput"),
  avatarPreview: document.querySelector("#avatarPreview"),
  photoFileInput: document.querySelector("#photoFileInput"),
  webcamPhotoButton: document.querySelector("#webcamPhotoButton"),
  clearPhotoButton: document.querySelector("#clearPhotoButton"),
  webcamBox: document.querySelector("#webcamBox"),
  webcamPreview: document.querySelector("#webcamPreview"),
  capturePhotoButton: document.querySelector("#capturePhotoButton"),
  cancelWebcamButton: document.querySelector("#cancelWebcamButton"),
  guestName: document.querySelector("#guestName"),
  guestRole: document.querySelector("#guestRole"),

  broadcasterPanel: document.querySelector("#broadcasterPanel"),
  broadcasterForm: document.querySelector("#broadcasterForm"),
  cameraNameInput: document.querySelector("#cameraNameInput"),
  videoDeviceInput: document.querySelector("#videoDeviceInput"),
  audioDeviceInput: document.querySelector("#audioDeviceInput"),
  broadcasterPreviewBox: document.querySelector("#broadcasterPreviewBox"),
  broadcasterPreview: document.querySelector("#broadcasterPreview"),
  previewButton: document.querySelector("#previewButton"),
  panelPublishButton: document.querySelector("#panelPublishButton"),
  deviceStatus: document.querySelector("#deviceStatus"),
  publishState: document.querySelector("#publishState"),

  adminPanel: document.querySelector("#adminPanel"),
  invitePanel: document.querySelector("#invitePanel"),
  eventForm: document.querySelector("#eventForm"),
  eventTitleInput: document.querySelector("#eventTitleInput"),
  eventDescriptionInput: document.querySelector("#eventDescriptionInput"),
  adminPasswordForm: document.querySelector("#adminPasswordForm"),
  guestPasswordInput: document.querySelector("#guestPasswordInput"),
  broadcasterPasswordInput: document.querySelector("#broadcasterPasswordInput"),
  adminPasswordInput: document.querySelector("#adminPasswordInput"),
  passwordStatus: document.querySelector("#passwordStatus"),
  refreshAdminButton: document.querySelector("#refreshAdminButton"),
  onlineCount: document.querySelector("#onlineCount"),
  viewerCount: document.querySelector("#viewerCount"),
  cameraCount: document.querySelector("#cameraCount"),
  liveCameraCount: document.querySelector("#liveCameraCount"),
  chatCameraCount: document.querySelector("#chatCameraCount"),
  viewerPresenceCount: document.querySelector("#viewerPresenceCount"),
  liveCameraList: document.querySelector("#liveCameraList"),
  cameraPresenceList: document.querySelector("#cameraPresenceList"),
  viewerPresenceList: document.querySelector("#viewerPresenceList"),
  journalList: document.querySelector("#journalList"),
  inviteForm: document.querySelector("#inviteForm"),
  inviteLabelInput: document.querySelector("#inviteLabelInput"),
  inviteRoleInput: document.querySelector("#inviteRoleInput"),
  inviteUsesInput: document.querySelector("#inviteUsesInput"),
  refreshInvitesButton: document.querySelector("#refreshInvitesButton"),
  inviteList: document.querySelector("#inviteList"),

  chatPanel: document.querySelector("#chatPanel"),
  chatStatus: document.querySelector("#chatStatus"),
  messages: document.querySelector("#messages"),
  chatForm: document.querySelector("#chatForm"),
  messageInput: document.querySelector("#messageInput")
};

boot();

function boot() {
  els.passwordTab.addEventListener("click", () => setAuthMode("password"));
  els.inviteTab.addEventListener("click", () => setAuthMode("invite"));
  els.loginForm.addEventListener("submit", onLogin);
  els.profileForm.addEventListener("submit", onProfileSave);
  els.photoFileInput.addEventListener("change", onPhotoFile);
  els.webcamPhotoButton.addEventListener("click", startPhotoWebcam);
  els.capturePhotoButton.addEventListener("click", capturePhoto);
  els.cancelWebcamButton.addEventListener("click", stopPhotoWebcam);
  els.clearPhotoButton.addEventListener("click", clearPhoto);
  els.broadcasterForm.addEventListener("submit", onBroadcasterSave);
  els.videoDeviceInput.addEventListener("change", onDeviceSelect);
  els.audioDeviceInput.addEventListener("change", onDeviceSelect);
  els.previewButton.addEventListener("click", startBroadcasterPreview);
  els.panelPublishButton.addEventListener("click", togglePublish);
  els.connectButton.addEventListener("click", connectLiveKit);
  els.publishButton.addEventListener("click", togglePublish);
  els.logoutButton.addEventListener("click", logout);
  els.chatForm.addEventListener("submit", onChatSubmit);
  els.eventForm.addEventListener("submit", onEventSave);
  els.adminPasswordForm.addEventListener("submit", onPasswordSave);
  els.refreshAdminButton.addEventListener("click", loadAdmin);
  els.inviteForm.addEventListener("submit", onInviteCreate);
  els.refreshInvitesButton.addEventListener("click", loadInvites);

  renderSession();
}

function renderSession() {
  const authenticated = Boolean(state.token && state.guest);
  els.loginPanel.hidden = authenticated;
  els.profilePanel.hidden = !authenticated;
  els.chatPanel.hidden = !authenticated;
  els.logoutButton.hidden = !authenticated;
  els.connectButton.disabled = !authenticated;
  els.publishButton.hidden = !authenticated || !canPublish();
  els.broadcasterPanel.hidden = !authenticated || !canPublish();
  els.adminPanel.hidden = !authenticated || state.guest.role !== "admin";
  els.invitePanel.hidden = !authenticated || state.guest.role !== "admin";

  if (!authenticated) {
    setStatus("Не подключено");
    return;
  }

  els.guestName.textContent = state.guest.name;
  els.guestRole.textContent = state.guest.role;
  els.profileNameInput.value = state.guest.name;
  els.cameraNameInput.value = state.guest.name;
  state.photoURL = state.guest.photo_url || "";
  renderAvatar();
  restoreDeviceSelection();
  if (canPublish()) {
    loadDevices();
  }
  updatePublishUI();
  setStatus("Вход выполнен");
  connectChat();

  if (state.guest.role === "admin") {
    loadAdmin();
  }
}

function setAuthMode(mode) {
  state.authMode = mode;
  const invite = mode === "invite";
  els.inviteTab.classList.toggle("active", invite);
  els.passwordTab.classList.toggle("active", !invite);
  els.inviteField.hidden = !invite;
  els.passwordField.hidden = invite;
  els.roleField.hidden = invite;
  els.loginError.textContent = "";
}

async function onLogin(event) {
  event.preventDefault();
  els.loginError.textContent = "";

  const name = els.nameInput.value.trim();
  if (!name) {
    els.loginError.textContent = "Введите имя";
    return;
  }

  try {
    const payload = state.authMode === "invite"
      ? await api("/api/guest/invite-login", {
          name,
          token: els.inviteInput.value.trim(),
          photo_url: state.photoURL
        })
      : await api("/api/guest/login", {
          name,
          password: els.passwordInput.value,
          role: els.roleInput.value,
          photo_url: state.photoURL
        });
    setSession(payload.guest, payload.token);
  } catch (error) {
    els.loginError.textContent = error.message;
  }
}

async function onProfileSave(event) {
  event.preventDefault();
  const payload = await api("/api/guest/profile", {
    name: els.profileNameInput.value.trim(),
    photo_url: state.photoURL
  });
  setSession(payload.guest, state.token);
}

async function onBroadcasterSave(event) {
  event.preventDefault();
  const name = els.cameraNameInput.value.trim();
  if (!name) return;
  const payload = await api("/api/guest/profile", {
    name,
    photo_url: state.photoURL
  });
  setSession(payload.guest, state.token);
}

function setSession(guest, token) {
  state.guest = guest;
  state.token = token;
  localStorage.setItem("guest", JSON.stringify(guest));
  localStorage.setItem("guest_token", token);
  renderSession();
}

async function connectLiveKit() {
  if (state.room) {
    await state.room.disconnect();
    state.room = null;
    clearVideos();
    setStatus("Отключено от видео");
    return;
  }

  try {
    const { token, url } = await api("/api/livekit/token", { can_publish: canPublish() });
    const room = new Room();
    state.room = room;
    wireRoom(room);
    setStatus("Подключение к видео");
    await room.connect(url, token);
    setStatus("Видео подключено");
    els.connectButton.textContent = "Отключиться";
    renderExistingTracks(room);
  } catch (error) {
    state.room = null;
    setStatus(error.message);
  }
}

function wireRoom(room) {
  room
    .on(RoomEvent.TrackSubscribed, (track, publication, participant) => {
      addTrack(track, participant.identity || participant.name || "camera");
    })
    .on(RoomEvent.TrackUnsubscribed, (track) => {
      removeTrack(track);
    })
    .on(RoomEvent.Disconnected, () => {
      state.room = null;
      void stopPublishing({ unpublish: false });
      clearVideos();
      els.connectButton.textContent = "Подключиться";
      setStatus("Видео отключено");
    });
}

function renderExistingTracks(room) {
  for (const participant of room.remoteParticipants.values()) {
    for (const publication of participant.trackPublications.values()) {
      if (publication.track) {
        addTrack(publication.track, participant.identity || participant.name || "camera");
      }
    }
  }
}

async function togglePublish() {
  if (!state.room) {
    await connectLiveKit();
  }
  if (!state.room) {
    return;
  }

  if (state.published) {
    await stopPublishing();
    return;
  }

  try {
    stopBroadcasterPreview();
    const tracks = await createLocalTracks(captureOptions());
    for (const track of tracks) {
      await state.room.localParticipant.publishTrack(track);
      if (track.kind === Track.Kind.Video) {
        addTrack(track, state.guest.name || "local", true);
      }
    }
    state.localTracks = tracks;
    state.published = true;
    updatePublishUI();
  } catch (error) {
    setStatus(error.message);
    els.deviceStatus.textContent = error.message;
  }
}

async function stopPublishing({ unpublish = true } = {}) {
  for (const track of state.localTracks) {
    if (unpublish && state.room) {
      await state.room.localParticipant.unpublishTrack(track).catch(() => {});
    }
    track.stop();
  }
  state.localTracks = [];
  state.published = false;
  removeLocalTiles();
  updatePublishUI();
}

function updatePublishUI() {
  const live = state.published;
  els.publishButton.textContent = live ? "Стоп" : "Камера";
  els.panelPublishButton.textContent = live ? "Остановить" : "В эфир";
  els.publishState.textContent = live ? "live" : "offline";
  els.previewButton.disabled = live;
  els.videoDeviceInput.disabled = live;
  els.audioDeviceInput.disabled = live;
  if (live) {
    els.deviceStatus.textContent = "Идет публикация камеры и микрофона";
  }
}

function captureOptions() {
  const videoDeviceId = els.videoDeviceInput.value;
  const audioDeviceId = els.audioDeviceInput.value;
  return {
    audio: audioDeviceId ? { deviceId: audioDeviceId } : true,
    video: videoDeviceId ? { deviceId: videoDeviceId } : true
  };
}

function mediaConstraints() {
  const options = captureOptions();
  return {
    audio: options.audio === true ? true : { deviceId: { exact: options.audio.deviceId } },
    video: options.video === true ? true : { deviceId: { exact: options.video.deviceId } }
  };
}

async function loadDevices() {
  if (!navigator.mediaDevices?.enumerateDevices) {
    els.deviceStatus.textContent = "Браузер не дает список устройств";
    return;
  }
  const devices = await navigator.mediaDevices.enumerateDevices().catch(() => []);
  renderDeviceOptions(els.videoDeviceInput, devices.filter((device) => device.kind === "videoinput"), "Камера");
  renderDeviceOptions(els.audioDeviceInput, devices.filter((device) => device.kind === "audioinput"), "Микрофон");
  restoreDeviceSelection();
}

function renderDeviceOptions(select, devices, fallbackLabel) {
  const selected = select.value;
  select.replaceChildren(new Option("Авто", ""));
  devices.forEach((device, index) => {
    select.append(new Option(device.label || `${fallbackLabel} ${index + 1}`, device.deviceId));
  });
  if ([...select.options].some((option) => option.value === selected)) {
    select.value = selected;
  }
}

async function startBroadcasterPreview() {
  stopBroadcasterPreview();
  try {
    state.previewStream = await navigator.mediaDevices.getUserMedia(mediaConstraints());
    els.broadcasterPreview.srcObject = state.previewStream;
    await els.broadcasterPreview.play();
    els.deviceStatus.textContent = "Предпросмотр включен";
    await loadDevices();
  } catch (error) {
    els.deviceStatus.textContent = error.message;
    setStatus(error.message);
  }
}

function stopBroadcasterPreview() {
  if (state.previewStream) {
    for (const track of state.previewStream.getTracks()) {
      track.stop();
    }
  }
  state.previewStream = null;
  els.broadcasterPreview.srcObject = null;
}

function onDeviceSelect() {
  localStorage.setItem("video_device_id", els.videoDeviceInput.value);
  localStorage.setItem("audio_device_id", els.audioDeviceInput.value);
  els.deviceStatus.textContent = "Выбор устройств сохранен";
}

function restoreDeviceSelection() {
  const videoDeviceId = localStorage.getItem("video_device_id") || "";
  const audioDeviceId = localStorage.getItem("audio_device_id") || "";
  if ([...els.videoDeviceInput.options].some((option) => option.value === videoDeviceId)) {
    els.videoDeviceInput.value = videoDeviceId;
  }
  if ([...els.audioDeviceInput.options].some((option) => option.value === audioDeviceId)) {
    els.audioDeviceInput.value = audioDeviceId;
  }
}

function addTrack(track, label, local = false) {
  removeEmptyState();
  const element = track.attach();
  const tile = document.createElement("div");
  tile.className = "tile";
  tile.dataset.sid = track.sid || track.mediaStreamTrack?.id || crypto.randomUUID();
  if (local) {
    tile.dataset.local = "true";
  }
  if (element.tagName === "VIDEO") {
    element.muted = local;
    element.playsInline = true;
  }
  const badge = document.createElement("div");
  badge.className = "tileLabel";
  badge.textContent = label;
  tile.append(element, badge);
  els.videoGrid.append(tile);
}

function removeTrack(track) {
  const sid = track.sid || track.mediaStreamTrack?.id;
  if (!sid) return;
  const tile = els.videoGrid.querySelector(`[data-sid="${CSS.escape(sid)}"]`);
  if (tile) tile.remove();
  ensureEmptyState();
}

function removeLocalTiles() {
  els.videoGrid.querySelectorAll("[data-local=true]").forEach((tile) => tile.remove());
  ensureEmptyState();
}

function clearVideos() {
  els.videoGrid.replaceChildren();
  ensureEmptyState();
}

function removeEmptyState() {
  els.videoGrid.querySelector(".emptyState")?.remove();
}

function ensureEmptyState() {
  if (els.videoGrid.children.length > 0) return;
  const empty = document.createElement("div");
  empty.className = "emptyState";
  empty.textContent = "Ожидание камер";
  els.videoGrid.append(empty);
}

function connectChat() {
  if (state.chat && state.chat.readyState <= WebSocket.OPEN) {
    return;
  }
  const protocol = location.protocol === "https:" ? "wss:" : "ws:";
  const socket = new WebSocket(`${protocol}//${location.host}/ws/chat?token=${encodeURIComponent(state.token)}`);
  state.chat = socket;

  socket.addEventListener("open", () => {
    els.chatStatus.textContent = "online";
  });
  socket.addEventListener("close", () => {
    els.chatStatus.textContent = "offline";
  });
  socket.addEventListener("message", (event) => {
    const envelope = readJSON(event.data);
    if (!envelope) return;
    if (envelope.type === "history") {
      els.messages.replaceChildren();
      for (const message of envelope.messages || []) {
        renderMessage(message);
      }
    }
    if (envelope.type === "message" && envelope.message) {
      renderMessage(envelope.message);
    }
  });
}

function onChatSubmit(event) {
  event.preventDefault();
  const text = els.messageInput.value.trim();
  if (!text || !state.chat || state.chat.readyState !== WebSocket.OPEN) {
    return;
  }
  state.chat.send(JSON.stringify({ text }));
  els.messageInput.value = "";
}

function renderMessage(message) {
  const row = document.createElement("div");
  row.className = "message";
  const avatar = document.createElement("img");
  avatar.className = "messageAvatar";
  avatar.alt = "";
  avatar.src = message.photo_url || avatarPlaceholder(message.name || "?");
  const body = document.createElement("div");
  body.className = "messageBody";
  const meta = document.createElement("div");
  meta.className = "messageMeta";
  meta.textContent = `${message.name || "Гость"} · ${new Date(message.created_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}`;
  const text = document.createElement("div");
  text.className = "messageText";
  text.textContent = message.text;
  body.append(meta, text);
  row.append(avatar, body);
  els.messages.append(row);
  els.messages.scrollTop = els.messages.scrollHeight;
}

async function onPhotoFile() {
  const file = els.photoFileInput.files?.[0];
  if (!file) return;
  state.photoURL = await imageFileToDataURL(file);
  renderAvatar();
}

async function startPhotoWebcam() {
  stopPhotoWebcam();
  try {
    state.photoStream = await navigator.mediaDevices.getUserMedia({ video: true, audio: false });
    els.webcamPreview.srcObject = state.photoStream;
    await els.webcamPreview.play();
    els.webcamBox.hidden = false;
  } catch (error) {
    setStatus(error.message);
  }
}

async function capturePhoto() {
  if (!state.photoStream) return;
  state.photoURL = videoFrameToDataURL(els.webcamPreview);
  renderAvatar();
  stopPhotoWebcam();
}

function stopPhotoWebcam() {
  if (state.photoStream) {
    for (const track of state.photoStream.getTracks()) {
      track.stop();
    }
  }
  state.photoStream = null;
  els.webcamPreview.srcObject = null;
  els.webcamBox.hidden = true;
}

function clearPhoto() {
  state.photoURL = "";
  els.photoFileInput.value = "";
  renderAvatar();
}

function renderAvatar() {
  const name = state.guest?.name || els.nameInput.value || "?";
  els.avatarPreview.src = state.photoURL || avatarPlaceholder(name);
}

async function imageFileToDataURL(file) {
  const bitmap = await createImageBitmap(file);
  const dataURL = imageToDataURL(bitmap, bitmap.width, bitmap.height);
  bitmap.close();
  return dataURL;
}

function videoFrameToDataURL(video) {
  return imageToDataURL(video, video.videoWidth, video.videoHeight);
}

function imageToDataURL(source, sourceWidth, sourceHeight) {
  const size = 240;
  const canvas = document.createElement("canvas");
  canvas.width = size;
  canvas.height = size;
  const ctx = canvas.getContext("2d");
  const scale = Math.max(size / sourceWidth, size / sourceHeight);
  const width = Math.round(sourceWidth * scale);
  const height = Math.round(sourceHeight * scale);
  const x = Math.round((size - width) / 2);
  const y = Math.round((size - height) / 2);
  ctx.fillStyle = "#dfe7eb";
  ctx.fillRect(0, 0, size, size);
  ctx.drawImage(source, x, y, width, height);
  return canvas.toDataURL("image/jpeg", 0.82);
}

async function onInviteCreate(event) {
  event.preventDefault();
  await api("/api/admin/invites", {
    role: els.inviteRoleInput.value,
    label: els.inviteLabelInput.value.trim(),
    max_uses: Number(els.inviteUsesInput.value || 0)
  });
  els.inviteLabelInput.value = "";
  await loadInvites();
}

async function onEventSave(event) {
  event.preventDefault();
  const payload = await api("/api/admin/event", {
    title: els.eventTitleInput.value.trim(),
    description: els.eventDescriptionInput.value.trim()
  });
  renderEvent(payload.event);
}

async function loadAdmin() {
  await Promise.all([loadEvent(), loadPasswords(), loadStatus(), loadJournal(), loadInvites()]);
}

async function loadEvent() {
  if (!state.token || state.guest?.role !== "admin") return;
  const payload = await apiGet("/api/admin/event");
  renderEvent(payload.event);
}

function renderEvent(event) {
  if (!event) return;
  els.eventTitleInput.value = event.title || "";
  els.eventDescriptionInput.value = event.description || "";
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
  const response = await api("/api/admin/passwords", payload);
  els.guestPasswordInput.value = "";
  els.broadcasterPasswordInput.value = "";
  els.adminPasswordInput.value = "";
  renderPasswords(response.passwords);
}

async function loadPasswords() {
  if (!state.token || state.guest?.role !== "admin") return;
  const payload = await apiGet("/api/admin/passwords");
  renderPasswords(payload.passwords);
}

function renderPasswords(passwords) {
  if (!passwords) return;
  const labels = [
    passwords.guest_configured ? "гости: задан" : "гости: нет",
    passwords.broadcaster_configured ? "камеры: задан" : "камеры: нет",
    passwords.admin_configured ? "админ: задан" : "админ: нет"
  ];
  els.passwordStatus.textContent = labels.join(" · ");
}

async function loadStatus() {
  if (!state.token || state.guest?.role !== "admin") return;
  const payload = await apiGet("/api/admin/status");
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

async function loadJournal() {
  if (!state.token || state.guest?.role !== "admin") return;
  const payload = await apiGet("/api/admin/journal");
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

async function loadInvites() {
  if (!state.token || state.guest?.role !== "admin") return;
  const payload = await apiGet("/api/admin/invites");
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
    await api("/api/admin/invites/disable", { token: invite.token });
    await loadInvites();
  });
  item.append(title, token, meta, disable);
  els.inviteList.append(item);
}

async function api(path, body) {
  const response = await fetch(path, {
    method: "POST",
    headers: headers(),
    body: JSON.stringify(body)
  });
  return parseResponse(response);
}

async function apiGet(path) {
  const response = await fetch(path, {
    headers: headers()
  });
  return parseResponse(response);
}

function headers() {
  const result = { "Content-Type": "application/json" };
  if (state.token) {
    result.Authorization = `Bearer ${state.token}`;
  }
  return result;
}

async function parseResponse(response) {
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(payload.error || `HTTP ${response.status}`);
  }
  return payload;
}

function logout() {
  stopPhotoWebcam();
  stopBroadcasterPreview();
  if (state.chat) {
    state.chat.close();
  }
  if (state.room) {
    state.room.disconnect();
  }
  localStorage.removeItem("guest");
  localStorage.removeItem("guest_token");
  state.token = "";
  state.guest = null;
  state.room = null;
  state.chat = null;
  state.photoURL = "";
  state.localTracks = [];
  state.published = false;
  els.messages.replaceChildren();
  clearVideos();
  renderSession();
}

function setStatus(text) {
  els.statusLine.textContent = text;
}

function canPublish() {
  return state.guest?.role === "broadcaster" || state.guest?.role === "admin";
}

function avatarPlaceholder(name) {
  const initial = [...String(name).trim()][0]?.toUpperCase() || "?";
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="96" height="96" viewBox="0 0 96 96"><rect width="96" height="96" rx="48" fill="#dfe7eb"/><text x="48" y="58" text-anchor="middle" font-family="system-ui, sans-serif" font-size="34" font-weight="700" fill="#66737c">${escapeXML(initial)}</text></svg>`;
  return `data:image/svg+xml;base64,${btoa(unescape(encodeURIComponent(svg)))}`;
}

function escapeXML(value) {
  return value.replace(/[<>&'"]/g, (char) => ({
    "<": "&lt;",
    ">": "&gt;",
    "&": "&amp;",
    "'": "&apos;",
    "\"": "&quot;"
  })[char]);
}

function readJSON(raw) {
  if (!raw) return null;
  try {
    return JSON.parse(raw);
  } catch {
    return null;
  }
}
