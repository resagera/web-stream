# Серверная часть

Этот файл фиксирует текущее состояние backend-реализации, чтобы по нему можно было быстро восстановить контекст проекта.

## Назначение

Backend является центральной точкой для семейной трансляции:

1. пускает гостей по общему паролю события;
2. хранит профиль гостя: имя, фото, служебный secret;
3. выдает токен для подключения к LiveKit;
4. держит WebSocket-чат;
5. хранит последние сообщения в памяти и пишет все сообщения на диск.

На первом этапе база данных не используется. Для ожидаемых 20 или меньше зрителей достаточно файлов в `server/data`.

## Текущая структура

```text
./
  PLAN.md                        # общий план работ по проекту
  about.md                       # исходное описание проекта
  Caddyfile                      # production TLS reverse proxy
  certs/.gitkeep                 # место для TURN/TLS сертификатов; приватные файлы игнорируются git
  docs/production.md             # notes по production-деплою
  docker-compose.yml             # backend + LiveKit
  livekit.yaml                   # reference/default конфиг LiveKit, RTC ports и опциональный TURN
  scripts/backup-data.sh         # архивирует server/data в backups/home-stream-data-*.tar.gz
  scripts/bootstrap-ubuntu.sh    # подготовка Ubuntu-сервера: Docker, Compose, .env, ufw
  scripts/livekit-entrypoint.sh   # рендерит runtime-конфиг LiveKit из env перед стартом контейнера
  scripts/restore-data.sh        # dry-run/apply восстановление backup-архива server/data
  scripts/smoke-local.sh          # локальная проверка compose/API/frontend на временном data-каталоге
  .env.example                   # пример переменных окружения
  server.md                      # этот файл

server/
  cmd/server/main.go              # точка входа, запуск HTTP-сервера и graceful shutdown

  internal/auth/token.go          # простой guest token: guest_id.secret
  internal/chat/hub.go            # список клиентов, история последних сообщений, broadcast
  internal/chat/message.go        # модель сообщения и WebSocket envelope
  internal/chat/storage.go        # JSONL-хранилище сообщений на диске
  internal/chat/websocket.go      # минимальный WebSocket upgrade/read/write на stdlib
  internal/config/config.go       # env-конфиг и пути к data-файлам
  internal/datafile/recover.go    # backup поврежденных runtime-файлов и атомарная запись JSON
  internal/event/event.go         # настройки события
  internal/event/store.go         # JSON-хранилище настроек события
  internal/http/handlers.go       # HTTP API handlers
  internal/http/response.go       # JSON responses, ошибки, CORS
  internal/http/routes.go         # регистрация маршрутов
  internal/http/server.go         # сборка зависимостей сервера
  internal/id/id.go               # crypto-random URL-safe id
  internal/invite/invite.go       # модель invite-ссылки
  internal/invite/store.go        # JSON-хранилище invite-ссылок
  internal/journal/journal.go      # модель события журнала
  internal/journal/store.go        # JSONL-журнал входов, чата и LiveKit-событий
  internal/livekit/token.go       # LiveKit-compatible JWT HS256 без внешних зависимостей
  internal/livekit/webhook.go     # LiveKit webhook event structs и auth validation
  internal/livestatus/store.go    # runtime-статус LiveKit участников и published tracks
  internal/media/store.go         # файловое хранилище фото профиля, лимит размера и проверка сигнатуры
  internal/passwords/store.go     # JSON-хранилище паролей ролей, статус без выдачи секретов
  internal/profile/profile.go     # модель гостя
  internal/profile/store.go       # JSON-хранилище гостей
  internal/web/web.go             # embed frontend static files
  internal/web/static/index.html  # основной экран приложения
  internal/web/static/admin.html  # отдельная страница админки
  internal/web/static/app.css     # стили frontend
  internal/web/static/app.js      # frontend логика: auth, LiveKit, chat, admin invites
  internal/web/static/admin.js    # standalone admin UI: login, event, passwords, invites, status, journal

  data/.gitkeep                   # каталог runtime-данных
  .dockerignore                   # исключает runtime/build artifacts из Docker context
  Dockerfile                      # сборка backend-контейнера
  go.mod                          # Go module, сейчас без внешних зависимостей
```

## Почему пока без внешних зависимостей

Сейчас сервер собирается полностью на стандартной библиотеке Go. Это удобно для первого рабочего слоя: не нужен интернет для `go mod download`, меньше движущихся частей, проще поднять на сервере.

Позже можно заменить ручной WebSocket на `gorilla/websocket` или `nhooyr.io/websocket`, а LiveKit JWT на официальный SDK, если появятся причины.

## Конфигурация

Для Docker Compose есть пример:

```bash
cp .env.example .env
```

Переменные окружения:

```text
ADDR=:8080
DATA_DIR=data
PUBLIC_ORIGIN=
SECURE_COOKIES=false
MAX_PHOTO_URL_BYTES=350000
DATA_HOST_DIR=./server/data
GUEST_PASSWORD=change-me
BROADCASTER_PASSWORD=
ADMIN_PASSWORD=

LIVEKIT_URL=
LIVEKIT_API_KEY=
LIVEKIT_API_SECRET=
LIVEKIT_ROOM=family-event

LIVEKIT_TURN_ENABLED=false
TURN_DOMAIN=turn.example.com
LIVEKIT_TURN_UDP_PORT=3478
LIVEKIT_TURN_TLS_PORT=5349
LIVEKIT_TURN_RELAY_RANGE_START=50101
LIVEKIT_TURN_RELAY_RANGE_END=50200
LIVEKIT_TURN_EXTERNAL_TLS=false
LIVEKIT_TURN_CERT_FILE=
LIVEKIT_TURN_KEY_FILE=
```

Важно: `GUEST_PASSWORD=change-me` является только dev-default. Для реального запуска пароль нужно задать явно.

`BROADCASTER_PASSWORD` и `ADMIN_PASSWORD` по умолчанию пустые при прямом запуске backend. Это означает, что войти как broadcaster/admin нельзя, пока пароль явно не задан. В Docker Compose для dev-запуска заданы defaults из `.env.example`.

Для production нужно задать:

```text
PUBLIC_ORIGIN=https://stream.example.com
SECURE_COOKIES=true
```

Для текущего dev-конфига LiveKit ключи такие:

```text
LIVEKIT_API_KEY=devkey
LIVEKIT_API_SECRET=secret
```

В Docker Compose эти значения передаются и backend, и LiveKit. Контейнер LiveKit рендерит runtime-конфиг через `scripts/livekit-entrypoint.sh`, поэтому production-ключи не нужно дублировать в `livekit.yaml`.

TURN выключен по умолчанию. Для production fallback в сложных сетях нужно задать `LIVEKIT_TURN_ENABLED=true`, `TURN_DOMAIN` и, если нужен TURN/TLS, пути `LIVEKIT_TURN_CERT_FILE`/`LIVEKIT_TURN_KEY_FILE` внутри контейнера, например `/etc/livekit-certs/turn.example.com.crt`.

Файлы данных:

```text
server/data/guests.json   # гости и их secrets
server/data/passwords.json # пароли ролей guest/broadcaster/admin
server/data/invites.json  # invite-ссылки
server/data/event.json    # настройки события
server/data/chat.jsonl    # чат, одна JSON-запись на строку
server/data/journal.jsonl # журнал входов, чата и LiveKit-событий
server/data/photos/       # файлы фото профиля
```

Если `guests.json`, `passwords.json`, `invites.json` или `event.json` невозможно распарсить как JSON при старте, файл переименовывается в `*.corrupt.<timestamp>`, а сервер создает новый файл с пустым/default состоянием. Для `chat.jsonl` и `journal.jsonl` отдельные битые строки пропускаются; если ломается само чтение файла, он тоже уходит в `*.corrupt.<timestamp>`.

`guests.json`, `passwords.json`, `invites.json` и `event.json` записываются атомарно: данные пишутся во временный файл в той же директории, синхронизируются на диск и затем заменяют целевой файл через `rename`.

## Docker Compose

Корневой `docker-compose.yml` поднимает:

1. `backend` - Go-сервис из `./server`;
2. `livekit` - LiveKit server; runtime-конфиг генерируется из `.env` через `scripts/livekit-entrypoint.sh`.

Команда запуска:

```bash
docker compose up --build
```

Порты:

```text
8080              backend HTTP API и WebSocket-чат, host-порт можно менять через BACKEND_PORT
7880              LiveKit HTTP/WebSocket
7881              LiveKit TCP RTC fallback
50000-50100/udp   LiveKit UDP media ports
3478/udp          optional LiveKit TURN UDP
5349/tcp          optional LiveKit TURN/TLS
50101-50200/udp   optional LiveKit TURN relay ports
```

Runtime-данные backend монтируются в `./server/data`. TURN/TLS сертификаты при необходимости кладутся в `./certs` и монтируются в LiveKit как `/etc/livekit-certs`.

Для локальных проверок можно переопределить каталог runtime-данных:

```bash
DATA_HOST_DIR=/tmp/home-stream-data docker compose up -d --build
```

По умолчанию используется `./server/data`.

Production profile добавляет Caddy:

```bash
docker compose --profile production up -d --build
```

Caddy слушает `80`, `443/tcp`, `443/udp` и проксирует:

```text
APP_DOMAIN      -> backend:8080
LIVEKIT_DOMAIN  -> livekit:7880
```

Подробности: `docs/production.md`.

## Ubuntu Bootstrap

Для подготовки чистого Ubuntu-сервера добавлен скрипт:

```bash
scripts/bootstrap-ubuntu.sh
```

Он устанавливает базовые пакеты, Docker Engine, Docker Compose plugin, создает `.env` из `.env.example`, проверяет `docker compose config` и выводит команду запуска. При `--ufw` открывает backend, LiveKit media ports и optional TURN-порты.

Опционально можно включить базовые правила firewall:

```bash
scripts/bootstrap-ubuntu.sh --ufw
```

Скрипт не перезаписывает существующий `.env`.

## Backup

Runtime backup делается скриптом:

```bash
scripts/backup-data.sh
```

По умолчанию архивы пишутся в `backups/` с именем `home-stream-data-<timestamp>.tar.gz`, хранятся последние 14 архивов, а `*.corrupt.*` файлы исключаются. Настройки:

```text
BACKUP_KEEP=14
BACKUP_INCLUDE_CORRUPT=false
BACKUP_DATA_DIR=server/data
```

Пример cron:

```cron
15 3 * * * cd /path/to/home-stream && BACKUP_KEEP=14 scripts/backup-data.sh >/dev/null
```

Если файлы в `server/data` принадлежат container user и не читаются host-пользователем, backup нужно запускать через `sudo` или поправить права/владельца volume.

Restore:

```bash
scripts/restore-data.sh backups/home-stream-data-YYYYMMDDTHHMMSSZ.tar.gz
scripts/restore-data.sh --apply backups/home-stream-data-YYYYMMDDTHHMMSSZ.tar.gz
```

Без `--apply` это dry-run. Перед реальной заменой `server/data` restore-скрипт создает pre-restore архив текущих данных в `backups/home-stream-data-before-restore-<timestamp>.tar.gz`.

## Smoke Test

Локальная проверка dev-контура:

```bash
scripts/smoke-local.sh
```

Скрипт поднимает отдельный compose project `home-stream-smoke` на `BACKEND_PORT=18080`, использует временный `DATA_HOST_DIR`, проверяет `/health`, логин гостя/камеры/админа, выдачу LiveKit JWT, admin API и отдачу `/`, `/admin.html`, `/admin.js`. Для этой проверки LiveKit запускается без публикации RTC-портов на host, чтобы не конфликтовать с уже занятыми UDP-портами. После проверки контейнеры smoke-проекта останавливаются, временные данные удаляются.

## Frontend

Frontend встроен в backend и доступен по корневому URL:

```text
GET /
GET /app.css
GET /app.js
GET /admin.html
GET /admin.js
GET /admin -> 307 /admin.html
```

Отдельная npm-сборка сейчас не нужна. В браузере `app.js` импортирует LiveKit client SDK через CDN:

```js
https://cdn.jsdelivr.net/npm/livekit-client/+esm
```

Текущий экран умеет:

1. входить по паролю;
2. входить по invite token;
3. сохранять имя профиля;
4. подключаться к LiveKit комнате;
5. показывать remote video tracks в сетке;
6. публиковать камеру/микрофон для ролей `broadcaster` и `admin`;
7. работать с WebSocket-чатом;
8. создавать и отключать invite-ссылки для роли `admin`;
9. загружать фото профиля из файла;
10. делать фото профиля с вебки;
11. показывать аватары в чате;
12. задавать имя камеры в режиме устройства-транслятора.
13. показывать admin-счетчики online/viewers/cameras;
14. редактировать название и описание события.
15. показывать admin-журнал последних входов, chat-подключений и LiveKit-событий.
16. выбирать камеру и микрофон для устройства-транслятора.
17. показывать локальный предпросмотр перед эфиром.
18. сохранять выбранные устройства ввода в `localStorage`.
19. в admin status UI отдельно показывать камеры в эфире, устройства-трансляторы онлайн и зрителей;
20. менять пароли ролей `guest`, `broadcaster`, `admin` в админ-панели.
21. открывать отдельную админку на `/admin.html`.

## API

### `GET /health`

Проверка живости сервера.

Ответ:

```json
{"status":"ok"}
```

### `POST /api/guest/login`

Вход гостя по общему паролю события.

Запрос:

```json
{
  "name": "Андрей",
  "password": "event-password",
  "photo_url": "",
  "role": "guest"
}
```

Поддерживаемые роли:

```text
guest         обычный зритель, не может публиковать видео
broadcaster   устройство-транслятор, может публиковать видео
admin         админ, может публиковать видео и позже будет управлять событием
```

Пароль проверяется по роли:

```text
guest         server/data/passwords.json или GUEST_PASSWORD при первом старте
broadcaster   server/data/passwords.json или BROADCASTER_PASSWORD при первом старте
admin         server/data/passwords.json или ADMIN_PASSWORD при первом старте
```

Ответ:

```json
{
  "guest": {
    "id": "...",
    "name": "Андрей",
    "photo_url": "",
    "secret": "...",
    "ip": "127.0.0.1",
    "created_at": "2026-06-29T00:00:00Z",
    "updated_at": "2026-06-29T00:00:00Z"
  },
  "token": "guest_id.secret"
}
```

Сервер также ставит cookie `guest_token`. Клиент может использовать cookie или передавать токен явно:

```text
Authorization: Bearer guest_id.secret
```

Для WebSocket также поддерживается query-параметр:

```text
/ws/chat?token=guest_id.secret
```

### `POST /api/guest/profile`

Обновление имени или фото гостя. Требует авторизации.

Запрос:

```json
{
  "name": "Андрей",
  "photo_url": "data:image/jpeg;base64,..."
}
```

### `POST /api/guest/invite-login`

Вход по invite token без общего пароля события.

Запрос:

```json
{
  "name": "Андрей",
  "token": "invite-token",
  "photo_url": ""
}
```

Роль гостя берется из invite. Если invite был создан для `broadcaster`, новый гость получит право публиковать видео в LiveKit.

### `GET /api/admin/invites`

Список invite-ссылок. Требует авторизации гостя с ролью `admin`.

Ответ:

```json
{
  "invites": []
}
```

### `GET /api/admin/event`

Получить настройки события. Требует роль `admin`.

Ответ:

```json
{
  "event": {
    "title": "Family Stream",
    "description": "",
    "updated_at": "2026-06-29T00:00:00Z"
  }
}
```

### `POST /api/admin/event`

Обновить настройки события. Требует роль `admin`.

Запрос:

```json
{
  "title": "Family Stream",
  "description": "Семейная трансляция"
}
```

### `GET /api/admin/passwords`

Статус паролей ролей. Требует роль `admin`. Сами пароли не возвращаются.

Ответ:

```json
{
  "passwords": {
    "guest_configured": true,
    "broadcaster_configured": true,
    "admin_configured": true,
    "updated_at": "2026-06-29T00:00:00Z"
  }
}
```

### `POST /api/admin/passwords`

Обновить один или несколько паролей ролей. Требует роль `admin`. Переданные пароли должны быть не короче 6 символов; отсутствующие поля не меняются.

Запрос:

```json
{
  "guest_password": "new-guest-password",
  "broadcaster_password": "new-camera-password",
  "admin_password": "new-admin-password"
}
```

### `GET /api/admin/status`

Runtime-статус подключений. Требует роль `admin`.

Ответ:

```json
{
  "online": [],
  "viewers": [],
  "cameras": [],
  "livekit_cameras": [],
  "online_count": 0,
  "viewer_count": 0,
  "camera_count": 0,
  "chat_camera_count": 0,
  "message_count": 0
}
```

`online`, `viewers`, `cameras` строятся по активным WebSocket-подключениям чата. `livekit_cameras` строится по LiveKit webhook-событиям published video tracks. `camera_count` считается по LiveKit tracks, `chat_camera_count` оставлен как вспомогательный счетчик по ролям.

### `GET /api/admin/journal`

Последние события журнала. Требует роль `admin`.

Ответ:

```json
{
  "entries": [
    {
      "id": "...",
      "type": "guest_login",
      "guest_id": "...",
      "name": "Андрей",
      "role": "guest",
      "ip": "127.0.0.1",
      "detail": "password",
      "created_at": "2026-06-29T00:00:00Z"
    }
  ]
}
```

Сейчас пишутся события `guest_login`, `profile_update`, `chat_connect`, `chat_disconnect` и LiveKit webhook-события с префиксом `livekit_`.

### `POST /api/livekit/webhook`

Внутренний endpoint для LiveKit webhooks. Сконфигурирован в `livekit.yaml`:

```yaml
webhook:
  api_key: devkey
  urls:
    - http://backend:8080/api/livekit/webhook
```

Обрабатываются события:

```text
participant_joined
participant_left
track_published
track_unpublished
```

### `POST /api/admin/invites`

Создать invite-ссылку. Требует роль `admin`.

Запрос:

```json
{
  "role": "guest",
  "label": "Friends",
  "max_uses": 0
}
```

`max_uses=0` означает без лимита. Для устройства-транслятора обычно нужен `role=broadcaster` и `max_uses=1`.

Ответ:

```json
{
  "invite": {
    "token": "...",
    "role": "guest",
    "label": "Friends",
    "active": true,
    "max_uses": 0,
    "used_count": 0
  }
}
```

### `POST /api/admin/invites/disable`

Отключить invite-ссылку. Требует роль `admin`.

Запрос:

```json
{
  "token": "invite-token"
}
```

### `POST /api/livekit/token`

Выдача токена подключения к LiveKit. Требует авторизации и заполненных `LIVEKIT_API_KEY`, `LIVEKIT_API_SECRET`, `LIVEKIT_ROOM`.

Запрос:

```json
{
  "can_publish": false
}
```

Поле `can_publish` пока оставлено для совместимости формы запроса, но backend ему не доверяет. Право публикации вычисляется только по сохраненной роли гостя:

```text
guest         canPublish=false
broadcaster   canPublish=true
admin         canPublish=true
```

Ответ:

```json
{
  "token": "jwt",
  "url": "wss://...",
  "room": "family-event"
}
```

### `GET /ws/chat`

WebSocket-чаты. Требует авторизации через cookie, `Authorization` или `?token=`.

После подключения сервер отправляет историю:

```json
{
  "type": "history",
  "messages": []
}
```

Клиент отправляет сообщение:

```json
{
  "text": "Всем привет!"
}
```

Сервер рассылает всем:

```json
{
  "type": "message",
  "message": {
    "id": "...",
    "guest_id": "...",
    "name": "Андрей",
    "text": "Всем привет!",
    "created_at": "2026-06-29T00:00:00Z"
  }
}
```

## Что уже реализовано

1. Самостоятельный Go-модуль в `./server`.
2. HTTP-сервер с graceful shutdown.
3. Авторизация гостя по общему паролю.
4. Хранение гостей в `data/guests.json`.
5. Cookie и bearer token для повторного входа.
6. Обновление профиля гостя.
7. LiveKit JWT HS256.
8. WebSocket upgrade без внешних зависимостей.
9. История последних 1000 сообщений в памяти.
10. Запись всех сообщений в `data/chat.jsonl`.
11. Dockerfile для backend-контейнера.
12. Docker Compose для backend + LiveKit.
13. Базовый `livekit.yaml` и entrypoint для runtime-конфига LiveKit из env.
14. Корневой `PLAN.md`.
15. Серверные роли `guest`, `broadcaster`, `admin`.
16. LiveKit publish grant вычисляется на сервере по роли, а не по клиентскому `can_publish`.
17. Invite-ссылки с ролью, лимитом использований и отключением.
18. Frontend MVP встроен в backend через Go `embed`.
19. Admin API для настроек события и runtime-статуса online/viewers/cameras.
20. Ubuntu bootstrap script для установки Docker/Compose и подготовки `.env`.
21. Production profile с Caddy TLS reverse proxy.
22. CORS whitelist, secure cookies и лимит размера `photo_url`.
23. Фото профиля сохраняются как файлы в `server/data/photos`, а в профиле хранится `/media/photos/...`.
24. LiveKit webhook status для published camera tracks.
25. Журнал событий в `server/data/journal.jsonl` и admin endpoint `/api/admin/journal`.
26. Специализированный экран устройства-транслятора с выбором камеры/микрофона, предпросмотром и отдельной кнопкой эфира.
27. Опциональная LiveKit TURN-конфигурация для production: UDP TURN, TURN/TLS, relay range, firewall notes.
28. Проверка сигнатуры загружаемых фото профиля для JPEG/PNG/WebP.
29. Recovery поврежденных runtime data-файлов через backup в `*.corrupt.<timestamp>` и старт с пустым/default состоянием.
30. Скрипт регулярного backup `server/data` с ротацией архивов.
31. Admin frontend разделяет `livekit_cameras`, chat cameras и viewers в отдельных статус-списках.
32. Атомарная запись JSON state-файлов через temp file, `fsync` и `rename`.
33. Restore-скрипт для backup-архивов с dry-run режимом и pre-restore backup.

## Проверка

Команды выполнялись указанной версией Go:

```bash
env GOCACHE=/tmp/home-stream-go-cache \
  /home/resager/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.7.linux-amd64/bin/go test ./...

env GOCACHE=/tmp/home-stream-go-cache \
  /home/resager/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.7.linux-amd64/bin/go build -buildvcs=false ./cmd/server
```

Обе проверки проходят.

Проверка Docker Compose:

```bash
docker compose config
docker compose build backend

BACKEND_PORT=18080 docker compose up -d --force-recreate
curl -sS http://127.0.0.1:18080/health
curl -sS -X POST http://127.0.0.1:18080/api/guest/login \
  -H 'Content-Type: application/json' \
  -d '{"name":"Compose Test","password":"change-me"}'
curl -sS -X POST http://127.0.0.1:18080/api/livekit/token \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <guest-token>' \
  -d '{"can_publish":false}'
docker compose down
```

Проверка production compose profile:

```bash
docker compose --profile production config
```

Результат:

- `docker compose config` проходит;
- backend image собирается;
- backend и LiveKit стартуют вместе;
- `/health` отвечает `{"status":"ok"}`;
- login endpoint создает гостя;
- `/api/livekit/token` возвращает JWT, `url` и `room`.

На этой машине host-порт `8080` уже был занят, поэтому для проверки использовался `BACKEND_PORT=18080`.

Проверка frontend:

```bash
env ADDR=127.0.0.1:18081 DATA_DIR=/tmp/home-stream-frontend-runtime \
  GUEST_PASSWORD=test-pass ADMIN_PASSWORD=admin-pass BROADCASTER_PASSWORD=broadcaster-pass \
  LIVEKIT_URL=ws://localhost:7880 LIVEKIT_API_KEY=devkey LIVEKIT_API_SECRET=secret \
  ./server

curl -sS -I http://127.0.0.1:18081/
curl -sS http://127.0.0.1:18081/app.js
curl -sS http://127.0.0.1:18081/health
```

Проверка frontend внутри Docker image:

```bash
BACKEND_PORT=18082 docker compose up -d --force-recreate
curl -sS http://127.0.0.1:18082/
curl -sS http://127.0.0.1:18082/health
docker compose down
```

Визуальная проверка layout:

```bash
chromium --headless --no-sandbox --disable-gpu --window-size=1366,900 \
  --screenshot=.tmp/screenshots/desktop-fresh.png http://127.0.0.1:18083/

chromium --headless --no-sandbox --disable-gpu --window-size=390,844 \
  --screenshot=.tmp/screenshots/mobile-fresh.png http://127.0.0.1:18083/
```

Проверено: desktop и mobile layout рендерятся без видимых наложений, скрытые панели не показываются до авторизации.

## Ближайшие следующие шаги

1. Проверить LiveKit JS SDK и новый экран транслятора в браузере с реальной камерой.
2. Проверить TURN на реальном домене с DNS и сертификатом.
3. Проверить backup/restore на production volume с реальными правами файлов.
