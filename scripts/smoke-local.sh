#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

project="${COMPOSE_PROJECT_NAME:-home-stream-smoke}"
backend_port="${BACKEND_PORT:-18080}"
tmp_data="$(mktemp -d "${TMPDIR:-/tmp}/home-stream-smoke-data.XXXXXX")"
override_file="$(mktemp "${TMPDIR:-/tmp}/home-stream-smoke-compose.XXXXXX.yml")"

cat >"${override_file}" <<'YAML'
services:
  livekit:
    ports: !reset []
YAML

cleanup() {
  COMPOSE_PROJECT_NAME="${project}" \
  BACKEND_PORT="${backend_port}" \
  DATA_HOST_DIR="${tmp_data}" \
    docker compose -f docker-compose.yml -f "${override_file}" down --remove-orphans >/dev/null 2>&1 || true
  rm -rf "${tmp_data}"
  rm -f "${override_file}"
}
trap cleanup EXIT

require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    echo "${name} is required" >&2
    exit 1
  fi
}

extract_json_string() {
  local key="$1"
  sed -n "s/.*\"${key}\"[[:space:]]*:[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p" | head -n 1
}

request_json() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local token="${4:-}"
  local headers=(-H "Content-Type: application/json")
  if [[ -n "${token}" ]]; then
    headers+=(-H "Authorization: Bearer ${token}")
  fi
  if [[ "${method}" == "GET" ]]; then
    curl -fsS "${headers[@]}" "http://127.0.0.1:${backend_port}${path}"
  else
    curl -fsS "${headers[@]}" -X "${method}" -d "${body}" "http://127.0.0.1:${backend_port}${path}"
  fi
}

wait_for_health() {
  for _ in {1..60}; do
    if curl -fsS "http://127.0.0.1:${backend_port}/health" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  echo "backend did not become healthy on port ${backend_port}" >&2
  return 1
}

require_command docker
require_command curl

if ! docker compose version >/dev/null 2>&1; then
  echo "docker compose plugin is required" >&2
  exit 1
fi

echo "Starting local smoke stack on backend port ${backend_port}..."
COMPOSE_PROJECT_NAME="${project}" \
BACKEND_PORT="${backend_port}" \
DATA_HOST_DIR="${tmp_data}" \
PUBLIC_ORIGIN="http://127.0.0.1:${backend_port}" \
SECURE_COOKIES=false \
GUEST_PASSWORD=smoke-guest-pass \
BROADCASTER_PASSWORD=smoke-camera-pass \
ADMIN_PASSWORD=smoke-admin-pass \
LIVEKIT_URL=ws://localhost:7880 \
LIVEKIT_API_KEY=devkey \
LIVEKIT_API_SECRET=secret \
LIVEKIT_ROOM=family-event \
  docker compose -f docker-compose.yml -f "${override_file}" up -d --build >/dev/null

wait_for_health

echo "Checking guest login..."
guest_token="$(
  request_json POST /api/guest/login '{"name":"Smoke Guest","password":"smoke-guest-pass","role":"guest"}' \
    | extract_json_string token
)"
if [[ -z "${guest_token}" ]]; then
  echo "guest login did not return token" >&2
  exit 1
fi

echo "Checking broadcaster LiveKit grant..."
broadcaster_token="$(
  request_json POST /api/guest/login '{"name":"Smoke Camera","password":"smoke-camera-pass","role":"broadcaster"}' \
    | extract_json_string token
)"
livekit_token="$(request_json POST /api/livekit/token '{"can_publish":false}' "${broadcaster_token}" | extract_json_string token)"
if [[ "${livekit_token}" != *.*.* ]]; then
  echo "LiveKit token has unexpected format" >&2
  exit 1
fi

echo "Checking admin APIs..."
admin_token="$(
  request_json POST /api/guest/login '{"name":"Smoke Admin","password":"smoke-admin-pass","role":"admin"}' \
    | extract_json_string token
)"
request_json GET /api/admin/status "" "${admin_token}" >/dev/null
request_json GET /api/admin/journal "" "${admin_token}" >/dev/null
request_json POST /api/admin/event '{"title":"Smoke Event","description":"Local smoke test"}' "${admin_token}" >/dev/null
request_json POST /api/admin/passwords '{"guest_password":"smoke-guest-pass-2"}' "${admin_token}" >/dev/null
request_json POST /api/admin/invites '{"role":"guest","label":"Smoke","max_uses":1}' "${admin_token}" >/dev/null

echo "Checking frontend files..."
curl -fsS "http://127.0.0.1:${backend_port}/" >/dev/null
curl -fsS "http://127.0.0.1:${backend_port}/admin.html" >/dev/null
curl -fsS "http://127.0.0.1:${backend_port}/admin.js" >/dev/null

echo "Smoke test passed."
