#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

config="$(
  LIVEKIT_CONFIG_ONLY=true \
  LIVEKIT_DUMP_CONFIG=true \
  LIVEKIT_API_KEY=prod-key \
  LIVEKIT_API_SECRET=prod-secret \
  LIVEKIT_TURN_ENABLED=true \
  LIVEKIT_TURN_DOMAIN=turn.example.com \
  LIVEKIT_TURN_UDP_PORT=3478 \
  LIVEKIT_TURN_TLS_PORT=5349 \
  LIVEKIT_TURN_RELAY_RANGE_START=50101 \
  LIVEKIT_TURN_RELAY_RANGE_END=50200 \
  LIVEKIT_TURN_EXTERNAL_TLS=true \
  LIVEKIT_TURN_CERT_FILE=/etc/livekit-certs/turn.example.com.crt \
  LIVEKIT_TURN_KEY_FILE=/etc/livekit-certs/turn.example.com.key \
    sh "${repo_root}/scripts/livekit-entrypoint.sh"
)"

require_line() {
  local expected="$1"
  if ! grep -Fqx "${expected}" <<<"${config}"; then
    echo "Expected LiveKit config line is missing: ${expected}" >&2
    echo "--- rendered config ---" >&2
    printf '%s\n' "${config}" >&2
    exit 1
  fi
}

require_line "  enabled: true"
require_line "  domain: 'turn.example.com'"
require_line "  udp_port: 3478"
require_line "  tls_port: 5349"
require_line "  relay_range_start: 50101"
require_line "  relay_range_end: 50200"
require_line "  external_tls: true"
require_line "  cert_file: '/etc/livekit-certs/turn.example.com.crt'"
require_line "  key_file: '/etc/livekit-certs/turn.example.com.key'"
require_line "  'prod-key': 'prod-secret'"
require_line "  api_key: 'prod-key'"
require_line "    - http://backend:8080/api/livekit/webhook"

quoted="$(
  LIVEKIT_CONFIG_ONLY=true \
  LIVEKIT_DUMP_CONFIG=true \
  LIVEKIT_API_KEY="key'with-quote" \
  LIVEKIT_API_SECRET="secret'with-quote" \
    sh "${repo_root}/scripts/livekit-entrypoint.sh"
)"

if ! grep -Fqx "  'key''with-quote': 'secret''with-quote'" <<<"${quoted}"; then
  echo "YAML single quote escaping failed" >&2
  echo "--- rendered config ---" >&2
  printf '%s\n' "${quoted}" >&2
  exit 1
fi

echo "LiveKit config verification passed."
