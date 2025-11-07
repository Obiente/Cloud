#!/bin/sh
set -e

# Substitute environment variables in datasource provisioning files
# Grafana provisioning doesn't support ${VAR} syntax, so we substitute manually

if [ -d /etc/grafana/provisioning ]; then
  # Find all YAML files in provisioning directory
  find /etc/grafana/provisioning -name "*.yml" -type f | while read file; do
    # Create temporary file
    tmpfile="${file}.tmp"
    
    # Substitute environment variables using sed
    # This works even if envsubst is not available
    sed \
      -e "s|\${GRAFANA_POSTGRES_HOST}|${GRAFANA_POSTGRES_HOST:-postgres}|g" \
      -e "s|\${GRAFANA_METRICS_DB_HOST}|${GRAFANA_METRICS_DB_HOST:-timescaledb}|g" \
      -e "s|\${POSTGRES_USER}|${POSTGRES_USER:-obiente-postgres}|g" \
      -e "s|\${POSTGRES_PASSWORD}|${POSTGRES_PASSWORD:-obiente-postgres}|g" \
      -e "s|\${METRICS_DB_USER}|${METRICS_DB_USER:-${POSTGRES_USER:-obiente-postgres}}|g" \
      -e "s|\${METRICS_DB_PASSWORD}|${METRICS_DB_PASSWORD:-${POSTGRES_PASSWORD:-obiente-postgres}}|g" \
      "$file" > "$tmpfile" && mv "$tmpfile" "$file"
  done
fi

# Execute the original Grafana entrypoint
exec /run.sh "$@"

