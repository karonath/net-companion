#!/usr/bin/env bash
# Build Net-Companion Lite : frontend Vite -> go:embed -> binaire unique.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "==> Build frontend (Vite)"
( cd "$root/frontend" && npm install && npm run build )

echo "==> Build backend (go:embed)"
( cd "$root/backend" && go build -o net-companion . )

echo "==> OK : backend/net-companion"

mkdir -p "$root/release"
cp "$root/backend/net-companion" "$root/release/net-companion"
echo "==> Prêt pour clé USB : release/net-companion"
