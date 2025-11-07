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

