#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_root="$(mktemp -d "${TMPDIR:-/tmp}/home-stream-backup-restore.XXXXXX")"

cleanup() {
  rm -rf "${tmp_root}"
}
trap cleanup EXIT

data_dir="${tmp_root}/data"
backup_dir="${tmp_root}/backups"
restore_target="${tmp_root}/restored"

mkdir -p "${data_dir}/photos" "${backup_dir}"
printf '{"guest":{"name":"Smoke Guest"}}\n' >"${data_dir}/guests.json"
printf '{"guest_password":"smoke-guest-pass"}\n' >"${data_dir}/passwords.json"
printf '{"title":"Smoke Event"}\n' >"${data_dir}/event.json"
printf '{"text":"hello"}\n' >"${data_dir}/chat.jsonl"
printf 'photo-bytes\n' >"${data_dir}/photos/avatar.jpg"
printf 'broken\n' >"${data_dir}/guests.json.corrupt.20260101T000000Z"

archive="$(
  BACKUP_DATA_DIR="${data_dir}" \
  BACKUP_KEEP=3 \
    "${repo_root}/scripts/backup-data.sh" "${backup_dir}"
)"

if [[ ! -f "${archive}" ]]; then
  echo "Backup archive was not created: ${archive}" >&2
  exit 1
fi

if tar -tzf "${archive}" | grep -q 'corrupt'; then
  echo "Corrupt files should be excluded by default" >&2
  exit 1
fi

"${repo_root}/scripts/restore-data.sh" "${archive}" "${restore_target}" >/dev/null
if [[ -e "${restore_target}" ]]; then
  echo "Dry-run restore created target directory" >&2
  exit 1
fi

"${repo_root}/scripts/restore-data.sh" --apply "${archive}" "${restore_target}" >/dev/null

for path in guests.json passwords.json event.json chat.jsonl photos/avatar.jpg; do
  if [[ ! -f "${restore_target}/${path}" ]]; then
    echo "Restored file is missing: ${path}" >&2
    exit 1
  fi
done

if [[ -e "${restore_target}/guests.json.corrupt.20260101T000000Z" ]]; then
  echo "Excluded corrupt file was restored unexpectedly" >&2
  exit 1
fi

if ! cmp -s "${data_dir}/guests.json" "${restore_target}/guests.json"; then
  echo "Restored guests.json differs from source" >&2
  exit 1
fi

include_archive="$(
  BACKUP_DATA_DIR="${data_dir}" \
  BACKUP_INCLUDE_CORRUPT=true \
  BACKUP_KEEP=3 \
    "${repo_root}/scripts/backup-data.sh" "${backup_dir}"
)"

if ! tar -tzf "${include_archive}" | grep -q 'guests.json.corrupt.20260101T000000Z'; then
  echo "BACKUP_INCLUDE_CORRUPT=true did not include corrupt files" >&2
  exit 1
fi

echo "Backup/restore verification passed."
