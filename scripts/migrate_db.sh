#!/usr/bin/env bash
set -e

source scripts/lib.sh || { echo "Are you at repo root?"; exit 1; }

usage() {
  cat <<EOUSAGE
Usage: $0 [up|down|force|version]
EOUSAGE
}

database_user="${DATABASE_USER:-root}"
database_password="${DATABASE_PASSWORD:-example}"
database_host="${DATABASE_HOST:-127.0.0.1}"
database_port="${DATABASE_PORT:-3306}"
database_name="${DATABASE_NAME:-fortune_db}"

# Redirect stderr to stdout for logging
case "$1" in
  up|down|force|version)
    if [[ -n "$database_password" ]]; then
      dsn="mysql://$database_user:$database_password@tcp($database_host:$database_port)/$database_name"
    else
      dsn="mysql://$database_user@tcp($database_host:$database_port)/$database_name"
    fi

    # Run migration
    migrate \
      -source file:etc/migrations \
      -database "$dsn" \
      "$@" 2>&1
    ;;
  *)
    usage
    exit 1
    ;;
esac
