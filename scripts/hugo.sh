#!/usr/bin/env bash
# hugo.sh — Hugo build wrapper with automatic post-build obfuscation.
#
# Runs Hugo (Extended, with SCSS support), then automatically obfuscates
# class names and IDs in the built output. No Node.js required.
#
# Usage:
#   ./scripts/hugo.sh                        # dev server
#   ./scripts/hugo.sh server                 # dev server (explicit)
#   ./scripts/hugo.sh --source src --minify  # production build
#
# Environment:
#   SKIP_OBFUSCATE=1  – skip post-build obfuscation
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
THEME_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Determine if this is a production build (not a server)
IS_PROD=true
for arg in "$@"; do
  case "$arg" in
    server|--server) IS_PROD=false ;;
  esac
done

# Run Hugo
echo "[hugo] building..."
(cd "$THEME_DIR" && hugo "$@")
HUGO_EXIT=$?

# Post-build obfuscation (only for production builds)
if [ "$IS_PROD" = true ] && [ "${SKIP_OBFUSCATE:-}" != "1" ]; then
  # Find the output directory from args or default to public/
  DEST="public"
  for i in $(seq 1 $#); do
    arg="${!i}"
    if [ "$arg" = "--destination" ] || [ "$arg" = "-d" ]; then
      next=$((i+1))
      DEST="${!next}"
      break
    fi
  done

  if [ -d "$THEME_DIR/$DEST" ] && [ -f "$THEME_DIR/go.mod" ]; then
    echo "[hugo] obfuscating..."
    (cd "$THEME_DIR" && go run ./cmd/obfuscate --verbose "$DEST") || true
  fi
fi

exit $HUGO_EXIT
