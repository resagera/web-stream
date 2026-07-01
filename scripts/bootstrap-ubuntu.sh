#!/usr/bin/env bash
set -euo pipefail

configure_ufw=0
if [[ "${1:-}" == "--ufw" ]]; then
  configure_ufw=1
fi

if [[ ! -f /etc/os-release ]]; then
  echo "Cannot detect OS: /etc/os-release is missing" >&2
  exit 1
fi

# shellcheck disable=SC1091
. /etc/os-release
if [[ "${ID:-}" != "ubuntu" ]]; then
  echo "This script expects Ubuntu, got: ${PRETTY_NAME:-unknown}" >&2
  exit 1
fi

if [[ "${EUID}" -eq 0 ]]; then
  echo "Run as a regular sudo-enabled user, not as root." >&2
  exit 1
fi

if ! command -v sudo >/dev/null 2>&1; then
  echo "sudo is required" >&2
  exit 1
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

echo "Installing base packages..."
sudo apt-get update
sudo apt-get install -y ca-certificates curl gnupg ufw git

if ! command -v docker >/dev/null 2>&1; then
  echo "Installing Docker Engine from official Docker apt repository..."
  sudo install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
    | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  sudo chmod a+r /etc/apt/keyrings/docker.gpg

  arch="$(dpkg --print-architecture)"
  codename="${VERSION_CODENAME}"
  echo "deb [arch=${arch} signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu ${codename} stable" \
    | sudo tee /etc/apt/sources.list.d/docker.list >/dev/null

  sudo apt-get update
  sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
else
  echo "Docker is already installed."
fi

if ! groups "${USER}" | grep -qw docker; then
  echo "Adding ${USER} to docker group..."
  sudo usermod -aG docker "${USER}"
  echo "You may need to log out and log in again before docker works without sudo."
fi

if [[ ! -f .env ]]; then
  echo "Creating .env from .env.example..."
  cp .env.example .env
  chmod 600 .env
else
  echo ".env already exists; leaving it unchanged."
fi

mkdir -p backups
chmod 700 backups

echo "Checking port listeners..."
check_tcp_port() {
  local port="$1"
  if ss -ltn "( sport = :${port} )" | tail -n +2 | grep -q .; then
    echo "WARNING: TCP port ${port} is already in use"
  else
    echo "TCP port ${port}: free"
  fi
}

check_tcp_port 80
check_tcp_port 443
check_tcp_port 8080
check_tcp_port 7880
check_tcp_port 7881
check_tcp_port 5349

if [[ "${configure_ufw}" -eq 1 ]]; then
  echo "Configuring ufw rules..."
  sudo ufw allow OpenSSH
  sudo ufw allow 80/tcp
  sudo ufw allow 443/tcp
  sudo ufw allow 443/udp
  sudo ufw allow 8080/tcp
  sudo ufw allow 7880/tcp
  sudo ufw allow 7881/tcp
  sudo ufw allow 3478/udp
  sudo ufw allow 5349/tcp
  sudo ufw allow 50000:50100/udp
  sudo ufw allow 50101:50200/udp
  sudo ufw --force enable
fi

echo "Checking docker compose config..."
docker compose config >/dev/null

echo "Checking backup script syntax..."
bash -n scripts/backup-data.sh
bash -n scripts/restore-data.sh
bash -n scripts/smoke-local.sh
bash -n scripts/verify-backup-restore.sh
bash -n scripts/verify-livekit-config.sh

cat <<'EOF'

Bootstrap complete.

Before production launch:
1. Edit .env and change all change-me secrets.
2. Set LIVEKIT_URL to the public wss:// URL behind your domain/TLS proxy.
3. For TURN fallback, set LIVEKIT_TURN_ENABLED=true and configure TURN_DOMAIN/TLS files.
4. If docker group membership was just added, log out and log back in.

Run:
  docker compose up -d --build

Manual data backup:
  scripts/backup-data.sh

Restore dry run:
  scripts/restore-data.sh backups/home-stream-data-YYYYMMDDTHHMMSSZ.tar.gz

Example daily cron entry:
  15 3 * * * cd /path/to/home-stream && BACKUP_KEEP=14 scripts/backup-data.sh >/dev/null

Optional firewall setup:
  scripts/bootstrap-ubuntu.sh --ufw
EOF
