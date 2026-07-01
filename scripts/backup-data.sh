#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/backup-data.sh [backup-dir]

Environment:
  BACKUP_KEEP        number of newest archives to keep, default: 14
  BACKUP_INCLUDE_CORRUPT=true to include *.corrupt.* files, default: false
  BACKUP_DATA_DIR    source data directory, default: ./server/data
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
data_dir="${BACKUP_DATA_DIR:-${repo_root}/server/data}"
backup_dir="${1:-${repo_root}/backups}"
keep="${BACKUP_KEEP:-14}"
include_corrupt="${BACKUP_INCLUDE_CORRUPT:-false}"

if ! [[ "${keep}" =~ ^[0-9]+$ ]]; then
  echo "BACKUP_KEEP must be a non-negative integer" >&2
  exit 1
fi

if [[ ! -d "${data_dir}" ]]; then
  echo "Data directory does not exist: ${data_dir}" >&2
  exit 1
fi
if ! find "${data_dir}" -type f ! -readable -print -quit | grep -q '^'; then
  :
else
  echo "Some files in ${data_dir} are not readable by $(id -un)." >&2
  echo "Run the backup with sudo or adjust the data directory ownership." >&2
  exit 1
fi

mkdir -p "${backup_dir}"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
archive="${backup_dir}/home-stream-data-${timestamp}.tar.gz"
tmp_archive="${archive}.tmp"

tar_args=(-C "$(dirname "${data_dir}")" --exclude='data/.gitkeep')

if [[ "${include_corrupt}" != "true" ]]; then
  tar_args+=(--exclude='data/*.corrupt.*')
fi

tar -czf "${tmp_archive}" "${tar_args[@]}" data
chmod 600 "${tmp_archive}"
mv "${tmp_archive}" "${archive}"

if [[ "${keep}" -gt 0 ]]; then
  mapfile -t old_archives < <(
    find "${backup_dir}" -maxdepth 1 -type f -name 'home-stream-data-*.tar.gz' -printf '%T@ %p\n' \
      | sort -rn \
      | awk -v keep="${keep}" 'NR > keep { sub(/^[^ ]+ /, ""); print }'
  )
  for old_archive in "${old_archives[@]}"; do
    rm -f "${old_archive}"
  done
fi

echo "${archive}"
