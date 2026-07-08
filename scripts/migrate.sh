#!/usr/bin/env bash
# Database migration runner using golang-migrate CLI.
set -euo pipefail

MIGRATIONS_DIR="${MIGRATIONS_DIR:-./migrations}"
DATABASE_URL="${DATABASE_URL:-postgres://gateway:gateway@localhost:5432/gateway?sslmode=disable}"

ACTION="${1:-up}"

case "$ACTION" in
  up)
    echo "Running migrations UP..."
    migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" up
    ;;
  down)
    echo "Running migrations DOWN..."
    migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" down 1
    ;;
  force)
    VERSION="${2:-}"
    if [ -z "$VERSION" ]; then
      echo "Usage: $0 force <version>"
      exit 1
    fi
    echo "Forcing migration version to $VERSION..."
    migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" force "$VERSION"
    ;;
  version)
    migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" version
    ;;
  *)
    echo "Usage: $0 {up|down|force <version>|version}"
    exit 1
    ;;
esac
