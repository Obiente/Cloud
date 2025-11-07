#!/bin/sh
set -e

# Substitute environment variables in datasource provisioning files
# Grafana provisioning doesn't support ${VAR} syntax, so we substitute manually

# Debug: Log environment variables (without passwords)
echo "Grafana entrypoint: Starting variable substitution..."
echo "GRAFANA_POSTGRES_HOST=${GRAFANA_POSTGRES_HOST:-postgres}"
echo "GRAFANA_METRICS_DB_HOST=${GRAFANA_METRICS_DB_HOST:-timescaledb}"
echo "POSTGRES_USER=${POSTGRES_USER:-obiente-postgres}"
echo "METRICS_DB_USER=${METRICS_DB_USER:-${POSTGRES_USER:-obiente-postgres}}"
echo "METRICS_DB_PASSWORD is ${METRICS_DB_PASSWORD:+SET}${METRICS_DB_PASSWORD:-NOT SET}"
echo "ALERT_EMAIL=${ALERT_EMAIL:-admin@example.com}"

# Configure Grafana SMTP only if SMTP_HOST is set
if [ -n "${SMTP_HOST}" ]; then
  echo "SMTP_HOST is configured, enabling Grafana SMTP..."
  export GF_SMTP_ENABLED="true"
  export GF_SMTP_HOST="${SMTP_HOST}"
  export GF_SMTP_PORT="${SMTP_PORT:-587}"
  export GF_SMTP_USER="${SMTP_USERNAME:-}"
  export GF_SMTP_PASSWORD="${SMTP_PASSWORD:-}"
  export GF_SMTP_FROM_ADDRESS="${SMTP_FROM_ADDRESS:-${ALERT_EMAIL:-admin@example.com}}"
  export GF_SMTP_FROM_NAME="${SMTP_FROM_NAME:-Grafana}"
  export GF_SMTP_SKIP_VERIFY="${SMTP_SKIP_TLS_VERIFY:-false}"
  
  # Map SMTP_USE_STARTTLS boolean to Grafana's STARTTLS policy
  if [ "${SMTP_USE_STARTTLS}" = "false" ] || [ "${SMTP_USE_STARTTLS}" = "0" ]; then
    export GF_SMTP_STARTTLS_POLICY="NoStartTLS"
  else
    export GF_SMTP_STARTTLS_POLICY="${GF_SMTP_STARTTLS_POLICY:-OpportunisticStartTLS}"
  fi
else
  echo "SMTP_HOST is not configured, SMTP will be disabled"
  # Don't set any GF_SMTP_* variables - Grafana will disable SMTP automatically
fi

if [ -d /etc/grafana/provisioning ]; then
  # Find all YAML files in provisioning directory (recursively)
  find /etc/grafana/provisioning -name "*.yml" -type f | while read file; do
    echo "Processing: $file"
    
    # Substitute environment variables using sed in-place
    # Handle nested defaults for METRICS_DB_USER and METRICS_DB_PASSWORD
    METRICS_USER="${METRICS_DB_USER:-${POSTGRES_USER:-obiente-postgres}}"
    METRICS_PASSWORD="${METRICS_DB_PASSWORD:-${POSTGRES_PASSWORD:-obiente-postgres}}"
    
    sed -i \
      -e "s|\${GRAFANA_POSTGRES_HOST}|${GRAFANA_POSTGRES_HOST:-postgres}|g" \
      -e "s|\${GRAFANA_METRICS_DB_HOST}|${GRAFANA_METRICS_DB_HOST:-timescaledb}|g" \
      -e "s|\${POSTGRES_USER}|${POSTGRES_USER:-obiente-postgres}|g" \
      -e "s|\${POSTGRES_PASSWORD}|${POSTGRES_PASSWORD:-obiente-postgres}|g" \
      -e "s|\${METRICS_DB_USER}|${METRICS_USER}|g" \
      -e "s|\${METRICS_DB_PASSWORD}|${METRICS_PASSWORD}|g" \
      -e "s|\${ALERT_EMAIL:-admin@example.com}|${ALERT_EMAIL:-admin@example.com}|g" \
      "$file"
    
    if [ $? -eq 0 ]; then
      echo "Successfully substituted variables in $file"
      
      # Validate alerting configuration: check for invalid relativeTimeRange in condition queries
      if echo "$file" | grep -q "alerting"; then
        # Check for condition queries (refId: B) with invalid relativeTimeRange (from: 0, to: 0)
        if grep -A 10 "refId: B" "$file" | grep -A 5 "relativeTimeRange:" | grep -q "from: 0"; then
          if grep -A 10 "refId: B" "$file" | grep -A 5 "relativeTimeRange:" | grep -q "to: 0"; then
            echo "WARNING: Found invalid relativeTimeRange (from: 0, to: 0) in condition query in $file"
            echo "  Condition queries (refId B) should not have relativeTimeRange"
            echo "  This may cause Grafana alerting provisioning to fail"
          fi
        fi
      fi
      
      # Debug: Show first few lines to verify substitution
      echo "First 10 lines after substitution:"
      head -10 "$file" || true
    else
      echo "ERROR: Failed to substitute variables in $file"
    fi
  done
else
  echo "WARNING: /etc/grafana/provisioning directory not found"
fi

echo "Grafana entrypoint: Variable substitution complete, starting Grafana..."

# Execute the original Grafana entrypoint
exec /run.sh "$@"

