# План проекта

Проект: небольшая семейная трансляция с несколькими камерами, LiveKit для видео, Go backend для авторизации, профилей и чата.

## Цель первого рабочего контура

Одна команда поднимает серверную часть:

```bash
docker compose up --build
```

После этого должны работать:

1. backend API;
2. LiveKit server;
3. выдача LiveKit JWT из backend;
4. вход гостя по паролю;
5. WebSocket-чат с сохранением сообщений на диск.

## Этапы

### 1. Backend foundation

Статус: сделано.

Состав:

- Go module в `server/`;
- HTTP API;
- гостевой вход по общему паролю;
- профиль гостя;
- LiveKit JWT;
- WebSocket-чат;
- файловое хранение гостей и чата.

### 2. Docker Compose + LiveKit

Статус: сделано.

Состав:

- `docker-compose.yml` в корне;
- `livekit.yaml` в корне;
- `.env.example` с настройками;
- volume для `server/data`;
- настраиваемый host-каталог runtime data через `DATA_HOST_DIR`;
- порты backend и LiveKit наружу;
- настраиваемый host-порт backend через `BACKEND_PORT`.

Критерий готовности:

- `docker compose config` проходит;
- `docker compose build backend` проходит;
- `BACKEND_PORT=18080 docker compose up -d --force-recreate` поднимает backend и LiveKit;
- `/health` отвечает;
- `/api/guest/login` создает гостя;
- `/api/livekit/token` выдает токен при заданных LiveKit ключах.
- `scripts/smoke-local.sh` поднимает отдельный smoke-compose проект с временным data-каталогом и проверяет backend/API/frontend без публикации LiveKit RTC-портов на host.

### 3. Роли доступа

Статус: сделано.

Нужно разделить:

- guest: смотреть видео, писать в чат;
- broadcaster: публиковать видео с камеры/микрофона;
- admin: управлять событием, ссылками, списком зрителей и камер.

Важно: сейчас `can_publish` в `/api/livekit/token` принимает клиентский input. Это годится только для dev-первого слоя. В production право публикации должно определяться сервером.

Сделано:

- роли `guest`, `broadcaster`, `admin` добавлены в модель гостя;
- `POST /api/guest/login` принимает `role`;
- пароль проверяется по роли;
- `POST /api/livekit/token` больше не доверяет клиентскому `can_publish`;
- LiveKit `canPublish` выдается только `broadcaster` и `admin`;
- добавлен тест на guest/broadcaster publish grant.
- добавлен admin API для события, invite-ссылок, статуса, журнала и паролей ролей.

### 4. Invite-ссылки

Статус: сделано.

Нужно:

- хранить invite-токены;
- поддержать ссылки для гостей;
- поддержать ссылки для устройств-трансляторов;
- уметь отключать/перевыпускать ссылки.

Сделано:

- invite-токены хранятся в `server/data/invites.json`;
- invite содержит `role`, `label`, `active`, `max_uses`, `used_count`;
- `POST /api/admin/invites` создает invite;
- `GET /api/admin/invites` показывает список invite;
- `POST /api/admin/invites/disable` отключает invite;
- `POST /api/guest/invite-login` создает гостя по invite без общего пароля;
- invite для `broadcaster` дает LiveKit publish grant через серверную роль;
- добавлен тест на одноразовый broadcaster invite.

### 5. Frontend MVP

Статус: частично сделано.

Экран гостя:

- вход по паролю или invite;
- имя и фото;
- сетка LiveKit камер;
- чат рядом с видео;
- адаптивная раскладка для телефона.

Экран устройства-транслятора:

- выбор камеры/микрофона;
- предпросмотр;
- старт/стоп публикации;
- имя камеры: улица, гостиная, кухня, телефон.

Сделано:

- frontend встроен в Go backend через `embed`;
- `/` отдает рабочий интерфейс без отдельной сборки npm;
- есть вход по паролю и invite;
- есть профиль с обновлением имени;
- есть WebSocket-чат с историей;
- есть LiveKit connect/disconnect;
- для `broadcaster` и `admin` есть старт/стоп публикации камеры и микрофона;
- для `admin` есть создание, список и отключение invite-ссылок;
- интерфейс адаптируется под мобильную ширину;
- Docker image отдает frontend и API.
- добавлено фото профиля через файл или снимок с вебки;
- сообщения чата показывают аватар;
- добавлен отдельный блок устройства-транслятора с именем камеры;
- экран устройства-транслятора умеет выбирать камеру и микрофон, показывать локальный предпросмотр и запускать/останавливать эфир;
- выбранные камера и микрофон сохраняются в браузере;
- desktop/mobile layout проверен headless Chromium screenshots.
- admin status UI разделяет камеры в эфире из LiveKit webhook, устройства-трансляторы онлайн по чату и зрителей онлайн.

Осталось:

- проверить LiveKit JS SDK в реальном браузере с камерой;
- проверить новый экран устройства-транслятора на реальном телефоне/ноутбуке с несколькими устройствами ввода.

### 6. Админка

Статус: сделано.

Минимум:

- логин админа;
- настройки события;
- общий пароль;
- invite-ссылки;
- список подключенных гостей;
- список активных камер.

Сделано:

- `GET /api/admin/event` читает настройки события из `server/data/event.json`;
- `POST /api/admin/event` обновляет название и описание события;
- `GET /api/admin/status` возвращает online/viewers/cameras по активным chat WebSocket подключениям;
- LiveKit webhook обновляет `livekit_cameras` по `track_published/track_unpublished`;
- frontend admin-панель показывает счетчики и список online;
- frontend admin-панель редактирует название и описание события.
- добавлен журнал подключений и системных событий в `server/data/journal.jsonl`;
- frontend admin-панель показывает последние события журнала.
- добавлено управление паролями ролей `guest`, `broadcaster`, `admin` через `GET/POST /api/admin/passwords`;
- пароли ролей сохраняются в `server/data/passwords.json`, а в ответах API показывается только статус наличия паролей;
- frontend admin-панель позволяет менять пароли без перезапуска backend.
- добавлена отдельная страница `/admin.html` и короткий redirect `/admin`;
- отдельная админка умеет входить как admin, менять событие и пароли, создавать/отключать invite, смотреть статус и журнал.

### 7. Production hardening

Статус: частично сделано.

Нужно:

- secure cookies за HTTPS;
- настройка CORS под домен;
- лимиты размера фото и сообщений;
- обработка поврежденных data-файлов;
- backup `server/data`;
- TURN/STUN-настройки для сложных сетей;
- инструкция деплоя на сервер с доменом и TLS.

Сделано:

- добавлен `Caddyfile` для TLS reverse proxy;
- добавлен production compose profile `caddy`;
- добавлены переменные `APP_DOMAIN`, `LIVEKIT_DOMAIN`, `ACME_EMAIL`;
- добавлены `PUBLIC_ORIGIN`, `SECURE_COOKIES`, `MAX_PHOTO_URL_BYTES`;
- добавлен `docs/production.md`;
- `livekit.yaml` переключен на `rtc.use_external_ip: true`;
- bootstrap script открывает `80/tcp`, `443/tcp`, `443/udp` при `--ufw`;
- `docker compose --profile production config` проходит.
- CORS может работать в strict whitelist режиме через `PUBLIC_ORIGIN`;
- guest cookie поддерживает `Secure`;
- `photo_url` ограничен по размеру.
- фото профиля сохраняются в `server/data/photos`, а не внутри `guests.json`.
- backend проверяет magic bytes для `jpeg`, `png`, `webp`, а не только MIME в data URL;
- compose поддерживает `DATA_HOST_DIR`, чтобы smoke/dev/prod могли использовать разные host-каталоги данных;
- поврежденные `guests.json`, `passwords.json`, `invites.json`, `event.json`, `chat.jsonl`, `journal.jsonl` переименовываются в `*.corrupt.<timestamp>`, сервер стартует с пустым/default состоянием;
- `guests.json`, `passwords.json`, `invites.json`, `event.json` пишутся атомарно через temp file + fsync + rename;
- добавлен `scripts/backup-data.sh` для ручного или cron backup `server/data` в `backups/home-stream-data-<timestamp>.tar.gz`;
- backup-скрипт поддерживает `BACKUP_KEEP`, `BACKUP_INCLUDE_CORRUPT`, `BACKUP_DATA_DIR` и проверяет читаемость файлов;
- добавлен `scripts/restore-data.sh` для dry-run/apply восстановления backup-архива с pre-restore backup текущих данных;
- добавлен `scripts/smoke-local.sh` для локальной проверки compose-контура без изменения `server/data`;
- добавлена опциональная LiveKit TURN-конфигурация: `turn.*` reference в `livekit.yaml`, env-переменные в `.env.example`, опубликованные TURN-порты в compose;
- добавлен `scripts/livekit-entrypoint.sh`, который рендерит runtime-конфиг LiveKit из `.env`, чтобы production API keys, webhook key и TURN не дублировались вручную;
- добавлен каталог `certs/` для TURN/TLS сертификата и ключа, приватные файлы игнорируются git;
- bootstrap script открывает `3478/udp`, `5349/tcp`, `50101:50200/udp` при `--ufw`.

Осталось:

- проверить TURN на реальном домене с DNS и сертификатом;
- проверить backup/restore на production volume с реальными правами файлов.

### 8. Ubuntu bootstrap script

Статус: сделано.

Нужно сделать скрипт автоматической подготовки Ubuntu-сервера:

- проверить версию Ubuntu и права пользователя;
- установить базовые пакеты: `ca-certificates`, `curl`, `gnupg`, `ufw`, `git`;
- установить Docker Engine и Docker Compose plugin из официального Docker apt repository;
- добавить текущего пользователя в группу `docker`;
- подготовить `.env` из `.env.example`, если `.env` еще нет;
- проверить доступность портов `8080`, `7880`, `7881`, `5349`;
- опционально настроить `ufw` под backend, LiveKit и SSH;
- проверить `docker compose config`;
- вывести финальную команду запуска `docker compose up -d --build`.

Файл планируемого скрипта:

```text
scripts/bootstrap-ubuntu.sh
```

Сделано:

- скрипт создан и помечен executable;
- проверяет Ubuntu и запуск не от root;
- ставит базовые пакеты;
- ставит Docker Engine и Compose plugin из официального Docker apt repository;
- добавляет пользователя в группу `docker`;
- создает `.env` из `.env.example`, если `.env` отсутствует;
- создает каталог `backups/`;
- проверяет занятость TCP-портов `8080`, `7880`, `7881`, `5349`;
- опционально настраивает `ufw` через `--ufw`;
- проверяет синтаксис `scripts/backup-data.sh`;
- проверяет синтаксис `scripts/restore-data.sh`;
- проверяет синтаксис `scripts/smoke-local.sh`;
- проверяет `docker compose config`;
- проходит `bash -n`.

## Текущий следующий шаг

Проверить LiveKit-подключение с реальной камерой в браузере, отдельную админку `/admin.html` в браузере и новый экран устройства-транслятора на реальном телефоне/ноутбуке. После этого проверить TURN на реальном домене с DNS и сертификатом.
