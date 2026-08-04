#!/usr/bin/env bash

set -Eeuo pipefail
umask 077

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly INSTALL_DIR="/usr/local/libexec/obiente-cloud"
readonly INSTALLED_SCRIPT="${INSTALL_DIR}/manage-preview-tls.sh"
readonly CONFIG_DIR="/etc/obiente"
readonly CONFIG_FILE="${CONFIG_DIR}/preview-tls.conf"
readonly SERVICE_FILE="/etc/systemd/system/obiente-preview-tls-renew.service"
readonly TIMER_FILE="/etc/systemd/system/obiente-preview-tls-renew.timer"

fail() {
  printf 'Error: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<'EOF'
Install automatic preview wildcard certificate renewal.

Usage:
  install-preview-tls-renewal.sh --provider NAME --credentials-file PATH \
    --email ADDRESS --accept-tos [options]

Required:
  --provider NAME              lego DNS provider code.
  --credentials-file PATH      Protected provider dotenv file.
  --email ADDRESS              ACME registration and recovery email.
  --accept-tos                 Accept the ACME certificate authority terms.

Options:
  --env-file PATH              Obiente Cloud .env file.
  --state-dir PATH             Persistent lego state directory.
  --stack-name NAME            Docker Swarm stack name (default: obiente).
  --lego-image IMAGE           Pinned lego image override.
  --ca-server URL_OR_NAME      Optional ACME CA server override.
  --renew-days DAYS            Renewal threshold (default: 30).
  --no-issue-now               Install the timer without initial issuance.
  --no-activate                Do not update an existing Traefik service.
  --help                       Show this help.

The timer runs daily with a randomized delay. lego performs no certificate or
secret rotation when the existing certificate is not due for renewal.
EOF
}

quote_systemd_environment_value() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  printf '"%s"' "$value"
}

write_config_value() {
  local key="$1"
  local value="$2"
  printf '%s=%s\n' "$key" "$(quote_systemd_environment_value "$value")" >> "$CONFIG_FILE"
}

main() {
  local provider=""
  local credentials_file=""
  local email=""
  local accept_tos="false"
  local env_file="${PREVIEW_TLS_ENV_FILE:-${SCRIPT_DIR}/../.env}"
  local state_dir="${PREVIEW_TLS_STATE_DIR:-/var/lib/obiente/preview-tls}"
  local stack_name="${PREVIEW_TLS_STACK_NAME:-obiente}"
  local lego_image="${PREVIEW_TLS_LEGO_IMAGE:-goacme/lego:v5.2.1}"
  local ca_server="${PREVIEW_TLS_CA_SERVER:-}"
  local renew_days="${PREVIEW_TLS_RENEW_DAYS:-30}"
  local issue_now="true"
  local activate="true"
  local credential_mode=""
  local -a validation_args=()

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --provider) [ "$#" -ge 2 ] || fail "--provider requires a value"; provider="$2"; shift 2 ;;
      --credentials-file) [ "$#" -ge 2 ] || fail "--credentials-file requires a value"; credentials_file="$2"; shift 2 ;;
      --email) [ "$#" -ge 2 ] || fail "--email requires a value"; email="$2"; shift 2 ;;
      --accept-tos) accept_tos="true"; shift ;;
      --env-file) [ "$#" -ge 2 ] || fail "--env-file requires a value"; env_file="$2"; shift 2 ;;
      --state-dir) [ "$#" -ge 2 ] || fail "--state-dir requires a value"; state_dir="$2"; shift 2 ;;
      --stack-name) [ "$#" -ge 2 ] || fail "--stack-name requires a value"; stack_name="$2"; shift 2 ;;
      --lego-image) [ "$#" -ge 2 ] || fail "--lego-image requires a value"; lego_image="$2"; shift 2 ;;
      --ca-server) [ "$#" -ge 2 ] || fail "--ca-server requires a value"; ca_server="$2"; shift 2 ;;
      --renew-days) [ "$#" -ge 2 ] || fail "--renew-days requires a value"; renew_days="$2"; shift 2 ;;
      --no-issue-now) issue_now="false"; shift ;;
      --no-activate) activate="false"; shift ;;
      -h|--help) usage; exit 0 ;;
      *) fail "Unknown option: $1" ;;
    esac
  done

  [ "$(id -u)" -eq 0 ] || fail "Run this installer as root"
  command -v systemctl >/dev/null 2>&1 || fail "systemctl is required"
  [ -x "${SCRIPT_DIR}/manage-preview-tls.sh" ] || fail "Missing ${SCRIPT_DIR}/manage-preview-tls.sh"
  [ -n "$provider" ] || fail "--provider is required"
  [ -n "$credentials_file" ] || fail "--credentials-file is required"
  [ -n "$email" ] || fail "--email is required"
  [ "$accept_tos" = "true" ] || fail "--accept-tos is required"

  [ -f "$credentials_file" ] || fail "Credentials file does not exist: $credentials_file"
  [ ! -L "$credentials_file" ] || fail "Credentials file must not be a symbolic link"
  credentials_file="$(realpath "$credentials_file")"
  credential_mode="$(stat -c '%a' "$credentials_file")"
  if (( (8#$credential_mode & 8#077) != 0 )); then
    fail "Credentials file must not be accessible by group or other users (mode $credential_mode)"
  fi
  env_file="$(realpath -m "$env_file")"
  state_dir="$(realpath -m "$state_dir")"

  validation_args=(
    check
    --provider "$provider"
    --credentials-file "$credentials_file"
    --email "$email"
    --env-file "$env_file"
    --state-dir "$state_dir"
    --stack-name "$stack_name"
    --lego-image "$lego_image"
    --renew-days "$renew_days"
    --accept-tos
  )
  [ -z "$ca_server" ] || validation_args+=(--ca-server "$ca_server")
  "${SCRIPT_DIR}/manage-preview-tls.sh" "${validation_args[@]}"

  install -d -m 0755 "$INSTALL_DIR"
  install -m 0755 "${SCRIPT_DIR}/manage-preview-tls.sh" "$INSTALLED_SCRIPT"
  install -d -m 0700 "$CONFIG_DIR"
  install -m 0600 "$credentials_file" "${CONFIG_DIR}/preview-dns-provider.env"
  : > "$CONFIG_FILE"
  chmod 0600 "$CONFIG_FILE"
  write_config_value PREVIEW_TLS_DNS_PROVIDER "$provider"
  write_config_value PREVIEW_TLS_DNS_CREDENTIALS_FILE "${CONFIG_DIR}/preview-dns-provider.env"
  write_config_value PREVIEW_TLS_EMAIL "$email"
  write_config_value PREVIEW_TLS_ENV_FILE "$env_file"
  write_config_value PREVIEW_TLS_STATE_DIR "$state_dir"
  write_config_value PREVIEW_TLS_STACK_NAME "$stack_name"
  write_config_value PREVIEW_TLS_LEGO_IMAGE "$lego_image"
  write_config_value PREVIEW_TLS_CA_SERVER "$ca_server"
  write_config_value PREVIEW_TLS_RENEW_DAYS "$renew_days"
  write_config_value PREVIEW_TLS_ACTIVATE "$activate"
  write_config_value PREVIEW_TLS_ACCEPT_TOS true

  install -m 0644 "${SCRIPT_DIR}/internal/systemd/obiente-preview-tls-renew.service" "$SERVICE_FILE"
  install -m 0644 "${SCRIPT_DIR}/internal/systemd/obiente-preview-tls-renew.timer" "$TIMER_FILE"
  systemctl daemon-reload

  if [ "$issue_now" = "true" ]; then
    systemctl start obiente-preview-tls-renew.service
  fi
  systemctl enable --now obiente-preview-tls-renew.timer

  printf 'Preview TLS renewal installed.\n'
  printf '  Configuration: %s\n' "$CONFIG_FILE"
  printf '  Timer status:  systemctl status obiente-preview-tls-renew.timer\n'
  printf '  Renewal logs:  journalctl -u obiente-preview-tls-renew.service\n'
}

main "$@"
