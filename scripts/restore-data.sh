#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/restore-data.sh [--apply] <archive.tar.gz> [data-dir]

Default mode is a dry run. Use --apply to replace the data directory.

Environment:
  RESTORE_BACKUP_DIR  where to put pre-restore backups, default: ./backups
EOF
}

apply=0
if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi
if [[ "${1:-}" == "--apply" ]]; then
  apply=1
  shift
fi
if [[ $# -lt 1 || $# -gt 2 ]]; then
  usage >&2
  exit 1
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
archive="$1"
data_dir="${2:-${repo_root}/server/data}"
backup_dir="${RESTORE_BACKUP_DIR:-${repo_root}/backups}"

if [[ ! -f "${archive}" ]]; then
  echo "Archive does not exist: ${archive}" >&2
  exit 1
fi

if ! tar -tzf "${archive}" >/dev/null; then
  echo "Archive is not a readable tar.gz: ${archive}" >&2
  exit 1
fi

if ! tar -tzf "${archive}" | grep -Eq '^data/'; then
  echo "Archive must contain a top-level data/ directory" >&2
  exit 1
fi

echo "Archive: ${archive}"
echo "Target data dir: ${data_dir}"
echo "Archive contents:"
tar -tzf "${archive}" | sed -n '1,40p'

if [[ "${apply}" -ne 1 ]]; then
  cat <<'EOF'

Dry run only. Re-run with --apply to restore:
  scripts/restore-data.sh --apply <archive.tar.gz> [data-dir]
EOF
  exit 0
fi

mkdir -p "${backup_dir}"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
pre_restore="${backup_dir}/home-stream-data-before-restore-${timestamp}.tar.gz"

if [[ -d "${data_dir}" ]]; then
  echo "Backing up current data to: ${pre_restore}"
  tar -czf "${pre_restore}.tmp" -C "$(dirname "${data_dir}")" "$(basename "${data_dir}")"
  chmod 600 "${pre_restore}.tmp"
  mv "${pre_restore}.tmp" "${pre_restore}"
fi

tmp_restore="$(mktemp -d /tmp/home-stream-restore.XXXXXX)"
trap 'rm -rf "${tmp_restore}"' EXIT
tar -xzf "${archive}" -C "${tmp_restore}"

mkdir -p "$(dirname "${data_dir}")"
rm -rf "${data_dir}.restore-new"
cp -a "${tmp_restore}/data" "${data_dir}.restore-new"
rm -rf "${data_dir}"
mv "${data_dir}.restore-new" "${data_dir}"

echo "Restore complete."
