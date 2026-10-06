#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

IMAGE="${IMAGE:-ghcr.io/benditandayusaputra/depotin-api:latest}"
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:8080/healthz}"

previous="$(docker compose images api --format '{{.ID}}' 2>/dev/null | head -n1 || true)"

echo "menarik citra $IMAGE"
docker compose pull api

echo "menjalankan migrasi"
docker compose run --rm --no-deps --entrypoint /app/migrate api up

echo "mengganti kontainer api"
docker compose up -d --no-deps api

echo "memeriksa kesehatan"
for attempt in $(seq 1 20); do
  if docker compose exec -T caddy wget -q -O - "http://api:8080/healthz" >/dev/null 2>&1; then
    echo "api sehat setelah ${attempt} percobaan"
    docker image prune -f >/dev/null
    exit 0
  fi
  sleep 3
done

echo "api tidak sehat, kembali ke citra sebelumnya"
if [ -n "$previous" ]; then
  docker tag "$previous" "$IMAGE"
  docker compose up -d --no-deps api
fi
exit 1
