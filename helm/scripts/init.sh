#!/bin/sh
# SQL Exporter Init Container Script
# This script handles dynamic configuration generation and bcrypt authentication setup

set -euo pipefail

# ECS-compliant logging function
log() {
  local level="$1"
  local message="$2"
  shift 2
  local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%S.%3NZ")
  
  # Build JSON log entry
  printf '{"@timestamp":"%s","log.level":"%s","message":"%s","ecs.version":"8.6.0","service.name":"sql-exporter-init"' \
    "$timestamp" "$level" "$message"
  
  # Add additional fields if provided
  while [ $# -gt 0 ]; do
    printf ',"%s":"%s"' "$1" "$2"
    shift 2
  done
  
  printf '}\n'
}

log_error() {
  log "error" "$@" >&2
}

log_info() {
  log "info" "$@"
}

log_debug() {
  log "debug" "$@"
}

# Dynamic config generation
if [ "${ENABLE_DYNAMIC_CONFIG:-false}" = "true" ]; then
  log_info "Starting dynamic config generation" "step" "dynamic_config" "enabled" "true"
  
  log_debug "Reading DSN from secret file" "step" "read_dsn" "file" "$PARTIAL_DSN_FILE"
  DSN=$(cat "$PARTIAL_DSN_FILE")
  
  # Mask password in DSN for logging (show only scheme and host)
  DSN_MASKED=$(echo "$DSN" | sed -E 's/(:[^:@]*@)/:*****@/g')
  log_debug "DSN loaded from secret" "step" "read_dsn" "dsn_masked" "$DSN_MASKED" "type" "$TYPE_SCHEME"
  
  # Add application_name to DSN if enabled
  if [ "${USE_APPLICATION_NAME:-false}" = "true" ] && [ -n "${APPLICATION_NAME:-}" ]; then
    log_debug "Adding application_name to DSN" "step" "add_app_name" "application_name" "$APPLICATION_NAME"
    if echo "$DSN" | grep -q '?'; then
      DSN="${DSN}&application_name=${APPLICATION_NAME}"
      log_debug "Appended application_name using '&'" "step" "add_app_name" "method" "append"
    else
      DSN="${DSN}?application_name=${APPLICATION_NAME}"
      log_debug "Appended application_name using '?'" "step" "add_app_name" "method" "new_query_string"
    fi
  fi
  
  log_debug "Decoding config template and substituting variables" "step" "template_substitution" "output_file" "/config-out/sql_exporter.yml"
  # Decode base64 template and substitute variables
  echo "$DYNAMIC_CONFIG_TEMPLATE_B64" | base64 -d | sed "s|__TYPE__|$TYPE_SCHEME|g; s|__DSN__|$DSN|g" > /config-out/sql_exporter.yml
  
  log_info "Dynamic config generation completed successfully" "step" "dynamic_config" "output_file" "/config-out/sql_exporter.yml"
fi

# Web config bcrypt generation
if [ "${ENABLE_BCRYPT:-false}" = "true" ]; then
  log_info "Starting web-config bcrypt generation" "step" "bcrypt_auth" "enabled" "true"
  
  HTPASSWD=/usr/local/apache2/bin/htpasswd
  log_debug "Reading password from secret" "step" "read_password" "secret_key" "$WC_PASSWORD_KEY"
  PASS=$(cat /secret-src/"$WC_PASSWORD_KEY")
  
  log_debug "Generating bcrypt hash" "step" "bcrypt_hash" "username" "$WC_USERNAME" "cost" "$WC_BCRYPT_COST" "htpasswd_path" "$HTPASSWD"
  HASH=$($HTPASSWD -nbBC "$WC_BCRYPT_COST" "$WC_USERNAME" "$PASS" | cut -d: -f2)
  
  if [ -z "$HASH" ]; then
    log_error "Bcrypt hash generation failed" "step" "bcrypt_hash" "username" "$WC_USERNAME"
    exit 1
  fi
  
  log_debug "Bcrypt hash generated successfully" "step" "bcrypt_hash" "hash_length" "${#HASH}"
  
  # Decode base64 web config template
  log_debug "Decoding web-config template" "step" "decode_template" "output_file" "/web-config/web-config.yml"
  echo "$WEBCONFIG_TEMPLATE_B64" | base64 -d > /web-config/web-config.yml
  
  # Append basic auth users
  log_debug "Appending basic auth user to web-config" "step" "append_auth" "username" "$WC_USERNAME"
  printf "\nbasic_auth_users:\n  %s: %s\n" "$WC_USERNAME" "$HASH" >> /web-config/web-config.yml
  
  log_info "Web-config bcrypt generation completed successfully" "step" "bcrypt_auth" "output_file" "/web-config/web-config.yml"
fi

log_info "Init container completed successfully" "step" "completion" "status" "success"

