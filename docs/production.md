# Production Deployment Notes

This project is designed to run behind a domain with HTTPS. The production compose profile adds Caddy as the public TLS reverse proxy.

## DNS

Create DNS records pointing to the server public IP:

```text
stream.example.com    A/AAAA    <server-ip>
livekit.example.com   A/AAAA    <server-ip>
turn.example.com      A/AAAA    <server-ip>  optional, for TURN fallback
```

Use two hostnames. The app hostname serves the frontend, backend API and chat WebSocket. The LiveKit hostname serves LiveKit's HTTP/WebSocket signaling endpoint.
If TURN is enabled, use a third hostname for TURN so clients receive a stable relay domain.

## Environment

Copy `.env.example` to `.env` and change all secrets:

```bash
cp .env.example .env
```

Required production values:

```text
APP_DOMAIN=stream.example.com
LIVEKIT_DOMAIN=livekit.example.com
ACME_EMAIL=you@example.com

LIVEKIT_URL=wss://livekit.example.com
LIVEKIT_API_KEY=<long-random-key>
LIVEKIT_API_SECRET=<long-random-secret>
PUBLIC_ORIGIN=https://stream.example.com
SECURE_COOKIES=true
MAX_PHOTO_URL_BYTES=350000

GUEST_PASSWORD=<event-password>
BROADCASTER_PASSWORD=<camera-password>
ADMIN_PASSWORD=<admin-password>
```

`LIVEKIT_API_KEY` and `LIVEKIT_API_SECRET` are passed to both backend and LiveKit. The LiveKit container renders its runtime config from `.env` via `scripts/livekit-entrypoint.sh`, so production keys do not need to be duplicated in `livekit.yaml`.

`PUBLIC_ORIGIN` enables a strict CORS allowlist. `SECURE_COOKIES=true` makes the guest cookie HTTPS-only.

LiveKit webhooks point to the backend over the private compose network:

```yaml
webhook:
  api_key: <LIVEKIT_API_KEY>
  urls:
    - http://backend:8080/api/livekit/webhook
```

## Start

Prepare Ubuntu:

```bash
scripts/bootstrap-ubuntu.sh --ufw
```

Start production profile:

```bash
docker compose --profile production up -d --build
```

Caddy listens on ports `80` and `443` and automatically obtains certificates. LiveKit media ports remain exposed directly:

```text
7881/tcp
50000-50100/udp
```

If TURN is enabled, these ports must also be reachable:

```text
3478/udp
5349/tcp
50101-50200/udp
```

## LiveKit RTC Notes

For a public server, `livekit.yaml` currently uses:

```yaml
rtc:
  tcp_port: 7881
  port_range_start: 50000
  port_range_end: 50100
  use_external_ip: true
```

This is usually enough for a small public VPS with the UDP range open. If guests are behind restrictive networks, enable the TURN configuration below.

## TURN

TURN is not enabled by default. For many family-event cases, direct UDP media ports plus TCP fallback are enough. If some guests cannot connect from corporate or restrictive networks, enable the built-in LiveKit TURN server.

The compose file already publishes the default TURN ports. `scripts/livekit-entrypoint.sh` renders these environment variables into the LiveKit runtime config:

```text
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

Minimal UDP TURN fallback:

```text
LIVEKIT_TURN_ENABLED=true
TURN_DOMAIN=turn.example.com
```

TURN/TLS fallback:

1. Put the certificate and private key under `certs/`, for example:

```text
certs/turn.example.com.crt
certs/turn.example.com.key
```

2. Set:

```text
LIVEKIT_TURN_ENABLED=true
TURN_DOMAIN=turn.example.com
LIVEKIT_TURN_CERT_FILE=/etc/livekit-certs/turn.example.com.crt
LIVEKIT_TURN_KEY_FILE=/etc/livekit-certs/turn.example.com.key
LIVEKIT_TURN_TLS_PORT=5349
```

3. Start the production profile again:

```bash
docker compose --profile production up -d --build
```

Notes:

- `certs/` is mounted read-only into the LiveKit container at `/etc/livekit-certs`.
- `certs/*` is ignored by git, because private keys must not be committed.
- Port `443` is already used by Caddy in this compose setup, so TURN/TLS uses `5349/tcp`.
- If you change `LIVEKIT_TURN_RELAY_RANGE_START` or `LIVEKIT_TURN_RELAY_RANGE_END`, update `docker-compose.yml` port mappings and firewall rules to the same range.

Do not reuse weak dev secrets for production LiveKit keys.

## Backup

Runtime data is in:

```text
server/data/
```

Back up these files:

```text
chat.jsonl
guests.json
invites.json
event.json
photos/
```

Create a manual archive:

```bash
scripts/backup-data.sh
```

By default archives are written to `backups/` and only the newest 14 archives are kept. The archive contains the `data/` directory. Old `*.corrupt.*` recovery files are excluded by default.

Useful overrides:

```bash
BACKUP_KEEP=30 scripts/backup-data.sh
BACKUP_INCLUDE_CORRUPT=true scripts/backup-data.sh
BACKUP_DATA_DIR=/path/to/data scripts/backup-data.sh /path/to/backups
```

Example daily cron entry:

```cron
15 3 * * * cd /path/to/home-stream && BACKUP_KEEP=14 scripts/backup-data.sh >/dev/null
```

If runtime files are owned by a container user and are not readable by the host user, run the backup through `sudo` or adjust ownership/permissions of `server/data`.

Restore outline:

```bash
docker compose down
tar -xzf backups/home-stream-data-YYYYMMDDTHHMMSSZ.tar.gz -C /tmp/home-stream-restore
rsync -a --delete /tmp/home-stream-restore/data/ server/data/
docker compose up -d
```
