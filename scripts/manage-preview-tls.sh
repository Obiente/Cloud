#!/usr/bin/env bash

set -Eeuo pipefail
umask 077

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly DEFAULT_LEGO_IMAGE="goacme/lego:v5.2.1"
readonly DEFAULT_PREVIEW_DOMAIN="my.obiente.cloud"
readonly DEFAULT_STATE_DIR="/var/lib/obiente/preview-tls"
readonly DEFAULT_RENEW_DAYS="30"
readonly DEFAULT_STACK_NAME="obiente"
readonly CERTIFICATE_NAME="preview-wildcard"

log() {
  printf '%s\n' "$*"
}

warn() {
  printf 'Warning: %s\n' "$*" >&2
}

fail() {
  printf 'Error: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<'EOF'
Manage the wildcard certificate used by Obiente Cloud pull request previews.

Usage:
  manage-preview-tls.sh setup [options]
  manage-preview-tls.sh renew [options]
  manage-preview-tls.sh status [options]
  manage-preview-tls.sh check [options]
  manage-preview-tls.sh bootstrap [options]

Commands:
  setup    Obtain the certificate when needed and install its Swarm secrets.
  renew    Renew when the configured threshold is reached and rotate secrets.
  status   Show the local certificate and active Traefik secret mappings.
  check    Validate configuration and credentials without issuing anything.
  bootstrap Create a seven-day self-signed certificate for first deployment.

Options:
  --accept-tos                 Accept the ACME certificate authority terms.
  --provider NAME              lego DNS provider code.
  --credentials-file PATH      Protected lego provider dotenv file.
  --email ADDRESS              ACME registration and recovery email.
  --env-file PATH              Obiente Cloud environment file to update.
  --state-dir PATH             Persistent lego account and certificate data.
  --stack-name NAME            Docker Swarm stack name (default: obiente).
  --lego-image IMAGE           Pinned lego container image.
  --ca-server URL_OR_NAME      Optional ACME CA server override.
  --renew-days DAYS            Renew at this many days remaining (default: 30).
  --issue-only                 Issue and validate without creating secrets.
  --no-activate                Do not update an existing Traefik service.
  --force                      Force certificate renewal and secret rotation.
  --help                       Show this help.

The same options can be supplied through PREVIEW_TLS_* environment variables.
Provider credentials are read from the mounted credentials file and are never
copied into .env or passed as Docker container environment variables.
EOF
}

trim_whitespace() {
  local value="$1"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  printf '%s' "$value"
}

is_allowed_deployment_env_key() {
  case "$1" in
    ACME_CA_SERVER|ACME_EMAIL|ENABLE_DNS|PREVIEW_ACME_CHALLENGE_CNAME|\
    PREVIEW_TLS_ACCEPT_TOS|PREVIEW_TLS_ACTIVATE|PREVIEW_TLS_CA_SERVER|\
    PREVIEW_TLS_CERT_SECRET|PREVIEW_TLS_DNS_CREDENTIALS_FILE|\
    PREVIEW_TLS_DNS_PROVIDER|PREVIEW_TLS_EMAIL|PREVIEW_TLS_KEY_SECRET|\
    PREVIEW_TLS_LEGO_IMAGE|PREVIEW_TLS_RENEW_DAYS|PREVIEW_TLS_STACK_NAME|\
    PREVIEW_TLS_STATE_DIR)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

load_env_file() {
  local env_file="$1"
  local line=""
  local key=""
  local value=""

  [ -f "$env_file" ] || return 0

  while IFS= read -r line || [ -n "$line" ]; do
    line="${line%$'\r'}"
    line="$(trim_whitespace "$line")"
    case "$line" in
      ""|\#*) continue ;;
    esac

    line="${line#export }"
    [[ "$line" == *=* ]] || continue
    key="$(trim_whitespace "${line%%=*}")"
    value="${line#*=}"
    value="${value#"${value%%[![:space:]]*}"}"

    [[ "$key" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]] || continue
    is_allowed_deployment_env_key "$key" || continue
    [[ -v "$key" ]] && continue
    if [[ "$value" == \"*\" && "$value" == *\" ]]; then
      value="${value:1:${#value}-2}"
    elif [[ "$value" == \'*\' && "$value" == *\' ]]; then
      value="${value:1:${#value}-2}"
    fi
    export "${key}=${value}"
  done < "$env_file"
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "Required command not found: $1"
}

validate_dns_name() {
  local name="$1"
  [[ "$name" =~ ^[a-z0-9]([a-z0-9.-]*[a-z0-9])?$ ]] || return 1
  [[ "$name" == *.* ]] || return 1
  [[ "$name" != *..* ]] || return 1
  [ "${#name}" -le 253 ] || return 1
}

validate_safe_name() {
  [[ "$1" =~ ^[A-Za-z0-9][A-Za-z0-9_.-]*$ ]]
}

validate_positive_integer() {
  [[ "$1" =~ ^[1-9][0-9]*$ ]]
}

validate_email() {
  [[ "$1" =~ ^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$ ]]
}

canonical_existing_file() {
  local path="$1"
  [ -f "$path" ] || fail "File does not exist: $path"
  [ ! -L "$path" ] || fail "Symbolic links are not accepted for protected files: $path"
  [[ "$path" != *,* ]] || fail "Protected file paths cannot contain commas: $path"
  realpath "$path"
}

canonical_directory() {
  local path="$1"
  [[ "$path" != *,* ]] || fail "State directory paths cannot contain commas: $path"
  mkdir -p "$path"
  chmod 700 "$path"
  realpath "$path"
}

validate_credentials_file() {
  local file="$1"
  local mode=""
  local line=""
  local key=""

  mode="$(stat -c '%a' "$file")"
  if (( (8#$mode & 8#077) != 0 )); then
    fail "Credentials file must not be accessible by group or other users: $file (mode $mode)"
  fi

  while IFS= read -r line || [ -n "$line" ]; do
    line="${line%$'\r'}"
    line="$(trim_whitespace "$line")"
    case "$line" in
      ""|\#*) continue ;;
    esac
    [[ "$line" == *=* ]] || fail "Invalid credentials entry; expected NAME=value"
    key="$(trim_whitespace "${line%%=*}")"
    [[ "$key" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]] || fail "Invalid credentials variable name: $key"
    case "$key" in
      LEGO_DEBUG_DNS_API_HTTP_CLIENT|LEGO_DEBUG_ACME_HTTP_CLIENT)
        fail "$key is not allowed because it can expose credentials in logs"
        ;;
    esac
  done < "$file"
}

validate_challenge_delegation() {
  local target="${PREVIEW_ACME_CHALLENGE_CNAME:-}"
  local normalized_target=""
  local label=""
  local -a labels=()

  case "${ENABLE_DNS:-true}" in
    false|0) return 0 ;;
  esac

  [ -n "$target" ] || fail "PREVIEW_ACME_CHALLENGE_CNAME is required while the bundled DNS service is enabled"
  normalized_target="${target,,}"
  normalized_target="${normalized_target%.}"
  [ "${#normalized_target}" -le 253 ] || fail "PREVIEW_ACME_CHALLENGE_CNAME is too long"
  [[ "$normalized_target" != *..* ]] || fail "PREVIEW_ACME_CHALLENGE_CNAME is not a valid DNS name"
  IFS='.' read -r -a labels <<< "$normalized_target"
  [ "${#labels[@]}" -ge 2 ] || fail "PREVIEW_ACME_CHALLENGE_CNAME must be a fully qualified DNS name"
  for label in "${labels[@]}"; do
    [ "${#label}" -le 63 ] || fail "PREVIEW_ACME_CHALLENGE_CNAME contains an oversized DNS label"
    [[ "$label" =~ ^[a-z0-9_]([a-z0-9_-]*[a-z0-9_])?$ ]] || fail "PREVIEW_ACME_CHALLENGE_CNAME is not a valid DNS name"
  done
  if [ "$normalized_target" = "$DEFAULT_PREVIEW_DOMAIN" ] ||
     [[ "$normalized_target" == *."$DEFAULT_PREVIEW_DOMAIN" ]]; then
    fail "PREVIEW_ACME_CHALLENGE_CNAME must point outside $DEFAULT_PREVIEW_DOMAIN"
  fi
}

verify_public_challenge_delegation() {
  local expected_target="${PREVIEW_ACME_CHALLENGE_CNAME:-}"
  local actual_target=""

  case "${ENABLE_DNS:-true}" in
    false|0) return 0 ;;
  esac
  command -v dig >/dev/null 2>&1 || return 0

  expected_target="${expected_target,,}"
  expected_target="${expected_target%.}."
  actual_target="$(dig +short CNAME "_acme-challenge.${DEFAULT_PREVIEW_DOMAIN}" 2>/dev/null | head -n 1 | tr '[:upper:]' '[:lower:]')"
  [ -n "$actual_target" ] || fail "The public ACME challenge CNAME is not available yet; deploy the DNS configuration before issuing"
  [ "$actual_target" = "$expected_target" ] || fail "Public ACME challenge CNAME points to $actual_target, expected $expected_target"
}

require_swarm_manager() {
  local swarm_state=""
  swarm_state="$(docker info --format '{{.Swarm.LocalNodeState}} {{.Swarm.ControlAvailable}}' 2>/dev/null || true)"
  [ "$swarm_state" = "active true" ] || fail "Run this command on a Docker Swarm manager"
}

certificate_fingerprint() {
  local certificate_file="$1"
  [ -f "$certificate_file" ] || return 0
  openssl x509 -in "$certificate_file" -noout -fingerprint -sha256 2>/dev/null | sed 's/^.*=//; s/://g'
}

validate_certificate_pair() {
  local certificate_file="$1"
  local key_file="$2"
  local domain="$3"
  local cert_public_key=""
  local private_public_key=""

  openssl x509 -in "$certificate_file" -noout >/dev/null 2>&1 || fail "lego produced an invalid certificate"
  openssl pkey -in "$key_file" -noout >/dev/null 2>&1 || fail "lego produced an invalid private key"
  openssl x509 -in "$certificate_file" -noout -checkend 86400 >/dev/null 2>&1 || fail "Certificate expires in less than 24 hours"
  openssl x509 -in "$certificate_file" -noout -checkhost "$domain" >/dev/null 2>&1 || fail "Certificate does not cover $domain"
  openssl x509 -in "$certificate_file" -noout -checkhost "preview-check.$domain" >/dev/null 2>&1 || fail "Certificate does not cover *.$domain"

  cert_public_key="$(openssl x509 -in "$certificate_file" -pubkey -noout 2>/dev/null | openssl pkey -pubin -outform DER 2>/dev/null | openssl dgst -sha256)"
  private_public_key="$(openssl pkey -in "$key_file" -pubout -outform DER 2>/dev/null | openssl dgst -sha256)"
  [ "$cert_public_key" = "$private_public_key" ] || fail "Certificate and private key do not match"
}

create_bootstrap_certificate() {
  local state_dir="$1"
  local env_file="$2"
  local stack_name="$3"
  local domain="$4"
  local bootstrap_dir="${state_dir}/bootstrap"
  local certificate_file="${bootstrap_dir}/preview-bootstrap.crt"
  local key_file="${bootstrap_dir}/preview-bootstrap.key"
  local fingerprint=""
  local secret_pair=""
  local certificate_secret=""
  local key_secret=""
  local service_name="${stack_name}_traefik"

  if docker service inspect "$service_name" >/dev/null 2>&1; then
    fail "Bootstrap certificates are only allowed before Traefik is deployed; use setup to rotate a live service"
  fi
  if docker secret inspect "${PREVIEW_TLS_CERT_SECRET:-preview_tls_cert}" >/dev/null 2>&1 ||
     docker secret inspect "${PREVIEW_TLS_KEY_SECRET:-preview_tls_key}" >/dev/null 2>&1; then
    fail "Selected preview TLS secrets already exist; bootstrap will not replace them"
  fi

  mkdir -p "$bootstrap_dir"
  chmod 700 "$bootstrap_dir"
  openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -days 7 \
    -keyout "$key_file" \
    -out "$certificate_file" \
    -subj "/CN=${domain}" \
    -addext "subjectAltName=DNS:${domain},DNS:*.${domain}" \
    >/dev/null 2>&1
  chmod 600 "$key_file"
  validate_certificate_pair "$certificate_file" "$key_file" "$domain"
  fingerprint="$(certificate_fingerprint "$certificate_file")"
  secret_pair="$(create_certificate_secrets "$certificate_file" "$key_file" "$fingerprint")"
  certificate_secret="${secret_pair%%|*}"
  key_secret="${secret_pair#*|}"
  write_env_secret_names "$env_file" "$certificate_secret" "$key_secret"

  export PREVIEW_TLS_CERT_SECRET="$certificate_secret"
  export PREVIEW_TLS_KEY_SECRET="$key_secret"
  log "Created a seven-day bootstrap certificate and selected its Swarm secrets."
  log "Deploy the stack, verify the public ACME challenge CNAME, then run setup immediately."
  show_status "$state_dir" "$env_file" "$stack_name"
}

write_env_secret_names() {
  local env_file="$1"
  local certificate_secret="$2"
  local key_secret="$3"
  local temp_file=""
  local env_dir=""

  env_dir="$(dirname "$env_file")"
  mkdir -p "$env_dir"
  temp_file="$(mktemp "${env_dir}/.preview-tls-env.XXXXXX")"

  if [ -f "$env_file" ]; then
    awk -v certificate_secret="$certificate_secret" -v key_secret="$key_secret" '
      BEGIN { wrote_certificate = 0; wrote_key = 0 }
      /^[[:space:]]*(export[[:space:]]+)?PREVIEW_TLS_CERT_SECRET[[:space:]]*=/ {
        if (!wrote_certificate) print "PREVIEW_TLS_CERT_SECRET=" certificate_secret
        wrote_certificate = 1
        next
      }
      /^[[:space:]]*(export[[:space:]]+)?PREVIEW_TLS_KEY_SECRET[[:space:]]*=/ {
        if (!wrote_key) print "PREVIEW_TLS_KEY_SECRET=" key_secret
        wrote_key = 1
        next
      }
      { print }
      END {
        if (!wrote_certificate) print "PREVIEW_TLS_CERT_SECRET=" certificate_secret
        if (!wrote_key) print "PREVIEW_TLS_KEY_SECRET=" key_secret
      }
    ' "$env_file" > "$temp_file"
    cp --attributes-only --preserve=all "$env_file" "$temp_file"
  else
    printf 'PREVIEW_TLS_CERT_SECRET=%s\nPREVIEW_TLS_KEY_SECRET=%s\n' "$certificate_secret" "$key_secret" > "$temp_file"
    chmod 600 "$temp_file"
  fi

  mv "$temp_file" "$env_file"
}

secret_source_for_target() {
  local service_name="$1"
  local target_name="$2"
  docker service inspect "$service_name" --format '{{range .Spec.TaskTemplate.ContainerSpec.Secrets}}{{printf "%s|%s\n" .SecretName .File.Name}}{{end}}' 2>/dev/null |
    awk -F '|' -v target="$target_name" '$2 == target { print $1; exit }'
}

activate_traefik_secrets() {
  local service_name="$1"
  local certificate_secret="$2"
  local key_secret="$3"
  local old_certificate_secret=""
  local old_key_secret=""
  local -a update_args=()

  if ! docker service inspect "$service_name" >/dev/null 2>&1; then
    log "Traefik service $service_name does not exist yet; the next stack deployment will use the new secrets."
    return 0
  fi

  old_certificate_secret="$(secret_source_for_target "$service_name" preview_tls_cert)"
  old_key_secret="$(secret_source_for_target "$service_name" preview_tls_key)"

  update_args=(docker service update --detach=false)
  [ -z "$old_certificate_secret" ] || update_args+=(--secret-rm "$old_certificate_secret")
  [ -z "$old_key_secret" ] || update_args+=(--secret-rm "$old_key_secret")
  update_args+=(
    --secret-add "source=${certificate_secret},target=preview_tls_cert,mode=0400"
    --secret-add "source=${key_secret},target=preview_tls_key,mode=0400"
    --force
    "$service_name"
  )

  log "Updating $service_name with the new wildcard certificate..."
  "${update_args[@]}"
}

create_certificate_secrets() {
  local certificate_file="$1"
  local key_file="$2"
  local fingerprint="$3"
  local suffix=""
  local certificate_secret=""
  local key_secret=""

  suffix="$(date -u +%Y%m%dT%H%M%SZ)_$(printf '%s' "$fingerprint" | cut -c1-12 | tr '[:upper:]' '[:lower:]')"
  certificate_secret="preview_tls_cert_${suffix}"
  key_secret="preview_tls_key_${suffix}"

  if ! docker secret create "$certificate_secret" "$certificate_file" >/dev/null; then
    fail "Failed to create certificate secret"
  fi
  if ! docker secret create "$key_secret" "$key_file" >/dev/null; then
    docker secret rm "$certificate_secret" >/dev/null 2>&1 || true
    fail "Failed to create private-key secret"
  fi

  printf '%s|%s\n' "$certificate_secret" "$key_secret"
}

write_pending_secret_pair() {
  local pending_file="$1"
  local fingerprint="$2"
  local certificate_secret="$3"
  local key_secret="$4"
  local temp_file=""

  temp_file="$(mktemp "$(dirname "$pending_file")/.preview-tls-pending.XXXXXX")"
  printf 'fingerprint=%s\ncertificate_secret=%s\nkey_secret=%s\n' \
    "$fingerprint" "$certificate_secret" "$key_secret" > "$temp_file"
  chmod 600 "$temp_file"
  mv "$temp_file" "$pending_file"
}

pending_secret_pair_for_fingerprint() {
  local pending_file="$1"
  local expected_fingerprint="$2"
  local line=""
  local fingerprint=""
  local certificate_secret=""
  local key_secret=""

  [ -f "$pending_file" ] || return 1
  [ ! -L "$pending_file" ] || return 1
  while IFS= read -r line || [ -n "$line" ]; do
    case "$line" in
      fingerprint=*) fingerprint="${line#*=}" ;;
      certificate_secret=*) certificate_secret="${line#*=}" ;;
      key_secret=*) key_secret="${line#*=}" ;;
      *) return 1 ;;
    esac
  done < "$pending_file"

  [[ "$fingerprint" =~ ^[A-Fa-f0-9]{64}$ ]] || return 1
  [ "${fingerprint^^}" = "${expected_fingerprint^^}" ] || return 1
  validate_safe_name "$certificate_secret" || return 1
  validate_safe_name "$key_secret" || return 1
  printf '%s|%s\n' "$certificate_secret" "$key_secret"
}

run_lego() {
  local state_dir="$1"
  local credentials_file="$2"
  local image="$3"
  local provider="$4"
  local email="$5"
  local domain="$6"
  local renew_days="$7"
  local ca_server="$8"
  local force_renewal="$9"
  local effective_renew_days="$renew_days"
  local -a lego_args=()
  local -a security_args=(--security-opt no-new-privileges:true)

  if [ "$force_renewal" = "true" ]; then
    effective_renew_days=36500
  fi

  lego_args=(
    --log.format text
    --path /lego
    --cert.name "$CERTIFICATE_NAME"
    --email "$email"
    --accept-tos
    --dns "$provider"
    --env-file /provider.env
    --domains "$domain"
    --domains "*.$domain"
    --renew-days "$effective_renew_days"
    --force-cert-domains
  )
  [ -z "$ca_server" ] || lego_args+=(--server "$ca_server")
  lego_args+=(run)

  if command -v getenforce >/dev/null 2>&1 && [ "$(getenforce 2>/dev/null || true)" != "Disabled" ]; then
    security_args+=(--security-opt label=disable)
  fi

  log "Obtaining or renewing the wildcard certificate with lego..."
  docker run --rm --pull missing \
    --user "$(id -u):$(id -g)" \
    --read-only \
    --tmpfs /tmp:rw,noexec,nosuid,size=16m \
    --cap-drop ALL \
    "${security_args[@]}" \
    --mount "type=bind,src=${state_dir},dst=/lego" \
    --mount "type=bind,src=${credentials_file},dst=/provider.env,readonly" \
    "$image" "${lego_args[@]}"
}

show_status() {
  local state_dir="$1"
  local env_file="$2"
  local stack_name="$3"
  local certificate_file="${state_dir}/certificates/${CERTIFICATE_NAME}.crt"
  local service_name="${stack_name}_traefik"

  log "Preview TLS status"
  log "  Environment file: $env_file"
  log "  State directory:   $state_dir"
  if [ -f "$certificate_file" ]; then
    log "  Certificate:       $certificate_file"
    openssl x509 -in "$certificate_file" -noout -subject -issuer -dates -fingerprint -sha256 | sed 's/^/    /'
  else
    log "  Certificate:       not issued locally"
  fi

  if docker secret inspect "${PREVIEW_TLS_CERT_SECRET:-preview_tls_cert}" >/dev/null 2>&1; then
    log "  Certificate secret: ${PREVIEW_TLS_CERT_SECRET:-preview_tls_cert}"
  else
    log "  Certificate secret: ${PREVIEW_TLS_CERT_SECRET:-preview_tls_cert} (missing)"
  fi
  if docker secret inspect "${PREVIEW_TLS_KEY_SECRET:-preview_tls_key}" >/dev/null 2>&1; then
    log "  Private-key secret: ${PREVIEW_TLS_KEY_SECRET:-preview_tls_key}"
  else
    log "  Private-key secret: ${PREVIEW_TLS_KEY_SECRET:-preview_tls_key} (missing)"
  fi

  if docker service inspect "$service_name" >/dev/null 2>&1; then
    log "  Traefik service:    $service_name"
    log "  Active cert source: $(secret_source_for_target "$service_name" preview_tls_cert)"
    log "  Active key source:  $(secret_source_for_target "$service_name" preview_tls_key)"
  else
    log "  Traefik service:    $service_name (not deployed)"
  fi
}

main() {
  local command_name="${1:-}"
  local cli_accept_tos=""
  local cli_provider=""
  local cli_credentials_file=""
  local cli_email=""
  local cli_env_file=""
  local cli_state_dir=""
  local cli_stack_name=""
  local cli_lego_image=""
  local cli_ca_server=""
  local cli_renew_days=""
  local cli_activate=""
  local issue_only="false"
  local force_renewal="false"
  local env_file=""
  local state_dir=""
  local provider=""
  local credentials_file=""
  local email=""
  local stack_name=""
  local domain=""
  local lego_image=""
  local ca_server=""
  local renew_days=""
  local activate=""
  local accept_tos=""
  local certificate_file=""
  local key_file=""
  local old_fingerprint=""
  local new_fingerprint=""
  local current_secrets_available="false"
  local secret_pair=""
  local new_certificate_secret=""
  local new_key_secret=""
  local env_backup=""
  local env_was_present="false"
  local service_name=""
  local active_certificate_secret=""
  local active_key_secret=""
  local pending_file=""

  case "$command_name" in
    setup|renew|status|check|bootstrap) shift ;;
    -h|--help|help|"") usage; exit 0 ;;
    *) fail "Unknown command: $command_name" ;;
  esac

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --accept-tos) cli_accept_tos="true"; shift ;;
      --provider) [ "$#" -ge 2 ] || fail "--provider requires a value"; cli_provider="$2"; shift 2 ;;
      --credentials-file) [ "$#" -ge 2 ] || fail "--credentials-file requires a value"; cli_credentials_file="$2"; shift 2 ;;
      --email) [ "$#" -ge 2 ] || fail "--email requires a value"; cli_email="$2"; shift 2 ;;
      --env-file) [ "$#" -ge 2 ] || fail "--env-file requires a value"; cli_env_file="$2"; shift 2 ;;
      --state-dir) [ "$#" -ge 2 ] || fail "--state-dir requires a value"; cli_state_dir="$2"; shift 2 ;;
      --stack-name) [ "$#" -ge 2 ] || fail "--stack-name requires a value"; cli_stack_name="$2"; shift 2 ;;
      --lego-image) [ "$#" -ge 2 ] || fail "--lego-image requires a value"; cli_lego_image="$2"; shift 2 ;;
      --ca-server) [ "$#" -ge 2 ] || fail "--ca-server requires a value"; cli_ca_server="$2"; shift 2 ;;
      --renew-days) [ "$#" -ge 2 ] || fail "--renew-days requires a value"; cli_renew_days="$2"; shift 2 ;;
      --issue-only) issue_only="true"; shift ;;
      --no-activate) cli_activate="false"; shift ;;
      --force) force_renewal="true"; shift ;;
      -h|--help) usage; exit 0 ;;
      *) fail "Unknown option: $1" ;;
    esac
  done

  env_file="${cli_env_file:-${PREVIEW_TLS_ENV_FILE:-}}"
  if [ -z "$env_file" ]; then
    if [ -f "${SCRIPT_DIR}/../.env" ]; then
      env_file="${SCRIPT_DIR}/../.env"
    else
      env_file="$(pwd)/.env"
    fi
  fi
  env_file="$(realpath -m "$env_file")"
  load_env_file "$env_file"

  state_dir="${cli_state_dir:-${PREVIEW_TLS_STATE_DIR:-$DEFAULT_STATE_DIR}}"
  provider="${cli_provider:-${PREVIEW_TLS_DNS_PROVIDER:-}}"
  credentials_file="${cli_credentials_file:-${PREVIEW_TLS_DNS_CREDENTIALS_FILE:-}}"
  email="${cli_email:-${PREVIEW_TLS_EMAIL:-${ACME_EMAIL:-}}}"
  stack_name="${cli_stack_name:-${PREVIEW_TLS_STACK_NAME:-$DEFAULT_STACK_NAME}}"
  domain="$DEFAULT_PREVIEW_DOMAIN"
  lego_image="${cli_lego_image:-${PREVIEW_TLS_LEGO_IMAGE:-$DEFAULT_LEGO_IMAGE}}"
  ca_server="${cli_ca_server:-${PREVIEW_TLS_CA_SERVER:-${ACME_CA_SERVER:-}}}"
  renew_days="${cli_renew_days:-${PREVIEW_TLS_RENEW_DAYS:-$DEFAULT_RENEW_DAYS}}"
  activate="${cli_activate:-${PREVIEW_TLS_ACTIVATE:-true}}"
  accept_tos="${cli_accept_tos:-${PREVIEW_TLS_ACCEPT_TOS:-false}}"

  domain="${domain,,}"
  validate_dns_name "$domain" || fail "Invalid preview domain: $domain"
  validate_safe_name "$stack_name" || fail "Invalid stack name: $stack_name"
  if ! [[ "$lego_image" =~ ^[A-Za-z0-9][A-Za-z0-9_./:@-]*$ ]]; then
    fail "Invalid lego image reference"
  fi
  validate_positive_integer "$renew_days" || fail "--renew-days must be a positive integer"
  case "$activate" in true|false) ;; *) fail "PREVIEW_TLS_ACTIVATE must be true or false" ;; esac
  case "$command_name" in
    setup|renew|check)
      if [[ "${ca_server,,}" == *staging* ]] && [ "$issue_only" != "true" ]; then
        fail "Staging ACME servers may only be used with --issue-only"
      fi
      ;;
  esac

  require_command docker
  require_command flock
  require_command openssl
  require_command realpath
  require_swarm_manager

  if [ "$issue_only" = "true" ]; then
    state_dir="${state_dir%/}/issue-only"
    force_renewal="true"
  fi
  state_dir="$(canonical_directory "$state_dir")"
  pending_file="${state_dir}/pending-secret-activation"
  exec 9>"${state_dir}/operation.lock"
  flock -n 9 || fail "Another preview TLS operation is already running"

  if [ "$command_name" = "status" ]; then
    show_status "$state_dir" "$env_file" "$stack_name"
    exit 0
  fi

  if [ "$command_name" = "bootstrap" ]; then
    create_bootstrap_certificate "$state_dir" "$env_file" "$stack_name" "$domain"
    exit 0
  fi

  validate_safe_name "$provider" || fail "A valid --provider is required"
  validate_email "$email" || fail "A valid --email is required"
  [ "$accept_tos" = "true" ] || fail "Pass --accept-tos after reviewing your ACME provider's terms"
  [ -n "$credentials_file" ] || fail "--credentials-file is required"
  credentials_file="$(canonical_existing_file "$credentials_file")"
  validate_credentials_file "$credentials_file"
  validate_challenge_delegation

  if [ "$command_name" = "check" ]; then
    log "Preview TLS configuration is valid. No certificate or deployment changes were made."
    exit 0
  fi

  certificate_file="${state_dir}/certificates/${CERTIFICATE_NAME}.crt"
  key_file="${state_dir}/certificates/${CERTIFICATE_NAME}.key"
  old_fingerprint="$(certificate_fingerprint "$certificate_file")"

  verify_public_challenge_delegation
  run_lego "$state_dir" "$credentials_file" "$lego_image" "$provider" "$email" "$domain" "$renew_days" "$ca_server" "$force_renewal"
  [ -f "$certificate_file" ] || fail "lego did not produce $certificate_file"
  [ -f "$key_file" ] || fail "lego did not produce $key_file"
  validate_certificate_pair "$certificate_file" "$key_file" "$domain"
  chmod 600 "$key_file"
  new_fingerprint="$(certificate_fingerprint "$certificate_file")"
  [ -n "$new_fingerprint" ] || fail "Could not read the issued certificate fingerprint"

  if [ "$issue_only" = "true" ]; then
    log "Certificate issuance and validation succeeded. No Swarm secrets or environment settings were changed."
    show_status "$state_dir" "$env_file" "$stack_name"
    exit 0
  fi

  if docker secret inspect "${PREVIEW_TLS_CERT_SECRET:-preview_tls_cert}" >/dev/null 2>&1 &&
     docker secret inspect "${PREVIEW_TLS_KEY_SECRET:-preview_tls_key}" >/dev/null 2>&1; then
    current_secrets_available="true"
  fi

  if [ -f "$pending_file" ]; then
    secret_pair="$(pending_secret_pair_for_fingerprint "$pending_file" "$new_fingerprint" || true)"
    if [ -n "$secret_pair" ]; then
      new_certificate_secret="${secret_pair%%|*}"
      new_key_secret="${secret_pair#*|}"
      if docker secret inspect "$new_certificate_secret" >/dev/null 2>&1 &&
         docker secret inspect "$new_key_secret" >/dev/null 2>&1; then
        log "Retrying the pending Traefik certificate activation."
      else
        warn "Discarding pending activation metadata because its Swarm secrets are missing."
        new_certificate_secret=""
        new_key_secret=""
        rm -f "$pending_file"
      fi
    else
      warn "Discarding stale or invalid pending activation metadata."
      rm -f "$pending_file"
    fi
  fi

  if [ -z "$new_certificate_secret" ] &&
     [ "$old_fingerprint" = "$new_fingerprint" ] &&
     [ "$current_secrets_available" = "true" ] &&
     [ "$force_renewal" = "false" ]; then
    service_name="${stack_name}_traefik"
    if [ "$activate" = "true" ] && docker service inspect "$service_name" >/dev/null 2>&1; then
      active_certificate_secret="$(secret_source_for_target "$service_name" preview_tls_cert)"
      active_key_secret="$(secret_source_for_target "$service_name" preview_tls_key)"
      if [ "$active_certificate_secret" != "${PREVIEW_TLS_CERT_SECRET:-preview_tls_cert}" ] ||
         [ "$active_key_secret" != "${PREVIEW_TLS_KEY_SECRET:-preview_tls_key}" ]; then
        log "The certificate is current, but Traefik has not loaded the selected secrets."
        if ! activate_traefik_secrets \
          "$service_name" \
          "${PREVIEW_TLS_CERT_SECRET:-preview_tls_cert}" \
          "${PREVIEW_TLS_KEY_SECRET:-preview_tls_key}"; then
          docker service rollback --detach=false "$service_name" >/dev/null 2>&1 || true
          fail "Traefik did not accept the selected preview TLS secrets"
        fi
      fi
    fi
    log "The certificate is not due for renewal; existing Swarm secrets remain active."
    show_status "$state_dir" "$env_file" "$stack_name"
    exit 0
  fi

  if [ -z "$new_certificate_secret" ]; then
    secret_pair="$(create_certificate_secrets "$certificate_file" "$key_file" "$new_fingerprint")"
    new_certificate_secret="${secret_pair%%|*}"
    new_key_secret="${secret_pair#*|}"
    write_pending_secret_pair "$pending_file" "$new_fingerprint" "$new_certificate_secret" "$new_key_secret"
    log "Created Swarm secrets $new_certificate_secret and $new_key_secret."
  fi

  env_backup="${state_dir}/env-backup-$(date -u +%Y%m%dT%H%M%SZ)"
  if [ -f "$env_file" ]; then
    env_was_present="true"
    cp --preserve=all "$env_file" "$env_backup"
  else
    : > "$env_backup"
    chmod 600 "$env_backup"
  fi
  write_env_secret_names "$env_file" "$new_certificate_secret" "$new_key_secret"

  service_name="${stack_name}_traefik"
  if [ "$activate" = "true" ]; then
    if ! activate_traefik_secrets "$service_name" "$new_certificate_secret" "$new_key_secret"; then
      warn "Traefik did not accept the new secret configuration; restoring the previous environment file."
      if [ "$env_was_present" = "true" ]; then
        cp --preserve=all "$env_backup" "$env_file"
      else
        rm -f "$env_file"
      fi
      docker service rollback --detach=false "$service_name" >/dev/null 2>&1 || true
      fail "Preview TLS activation failed; the new versioned secrets were retained for diagnosis"
    fi
  else
    log "Activation skipped. Deploy the stack when you are ready to load the new certificate."
  fi

  rm -f "$pending_file"
  export PREVIEW_TLS_CERT_SECRET="$new_certificate_secret"
  export PREVIEW_TLS_KEY_SECRET="$new_key_secret"
  log "Preview TLS certificate setup completed."
  log "Previous versioned secrets were retained for rollback."
  show_status "$state_dir" "$env_file" "$stack_name"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
