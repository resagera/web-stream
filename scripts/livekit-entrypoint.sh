#!/bin/sh
set -eu

yaml_quote() {
  printf "'%s'" "$(printf "%s" "$1" | sed "s/'/''/g")"
}

livekit_api_key="$(yaml_quote "${LIVEKIT_API_KEY:-devkey}")"
livekit_api_secret="$(yaml_quote "${LIVEKIT_API_SECRET:-secret}")"
turn_domain="$(yaml_quote "${LIVEKIT_TURN_DOMAIN:-${TURN_DOMAIN:-}}")"
turn_cert_file="$(yaml_quote "${LIVEKIT_TURN_CERT_FILE:-}")"
turn_key_file="$(yaml_quote "${LIVEKIT_TURN_KEY_FILE:-}")"

cat >/tmp/livekit.yaml <<EOF
port: 7880
bind_addresses:
  - ""

rtc:
  tcp_port: 7881
  port_range_start: 50000
  port_range_end: 50100
  use_external_ip: true

turn:
  enabled: ${LIVEKIT_TURN_ENABLED:-false}
  domain: ${turn_domain}
  udp_port: ${LIVEKIT_TURN_UDP_PORT:-3478}
  tls_port: ${LIVEKIT_TURN_TLS_PORT:-5349}
  relay_range_start: ${LIVEKIT_TURN_RELAY_RANGE_START:-50101}
  relay_range_end: ${LIVEKIT_TURN_RELAY_RANGE_END:-50200}
  external_tls: ${LIVEKIT_TURN_EXTERNAL_TLS:-false}
  cert_file: ${turn_cert_file}
  key_file: ${turn_key_file}
  ttl_seconds: 300

keys:
  ${livekit_api_key}: ${livekit_api_secret}

webhook:
  api_key: ${livekit_api_key}
  urls:
    - http://backend:8080/api/livekit/webhook

logging:
  level: info
EOF

if [ "${LIVEKIT_DUMP_CONFIG:-false}" = "true" ]; then
  cat /tmp/livekit.yaml
fi

exec /livekit-server --config /tmp/livekit.yaml "$@"
