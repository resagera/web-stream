# web-stream

Family streaming app with a Go backend, embedded frontend, WebSocket chat, LiveKit video rooms, Docker Compose deployment, Caddy TLS proxy, optional TURN fallback, and file-based runtime storage.

Main docs:

- `about.md` - project basis and product notes.
- `PLAN.md` - implementation plan and current state.
- `server.md` - backend structure, API, deployment notes, and verification commands.
- `docs/production.md` - production deployment, TLS, TURN, backup and restore notes.

Development checks:

```bash
cd server
env GOCACHE=/tmp/home-stream-go-cache /home/resager/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.7.linux-amd64/bin/go test ./...
env GOCACHE=/tmp/home-stream-go-cache /home/resager/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.7.linux-amd64/bin/go build -buildvcs=false ./cmd/server
```

Local compose:

```bash
cp .env.example .env
docker compose up -d --build
```
