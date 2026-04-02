#!/usr/bin/env bash
# Простой нагрузочный прогон для GET /rooms/{roomId}/slots/list (см. README).
# Требования: curl, jq, hey (https://github.com/rakyll/hey)
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
ROOM_ID="${ROOM_ID:-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa}"
DATE="${DATE:-$(date -u +%Y-%m-%d)}"
REQUESTS="${REQUESTS:-300}"
CONCURRENCY="${CONCURRENCY:-15}"

TOKEN_JSON=$(curl -fsS "${BASE_URL}/dummyLogin" \
  -H 'Content-Type: application/json' \
  -d '{"role":"user"}')
TOKEN=$(echo "${TOKEN_JSON}" | jq -r .token)

echo "BASE_URL=${BASE_URL} ROOM_ID=${ROOM_ID} DATE=${DATE} requests=${REQUESTS} concurrency=${CONCURRENCY}"
exec hey -n "${REQUESTS}" -c "${CONCURRENCY}" \
  -H "Authorization: Bearer ${TOKEN}" \
  "${BASE_URL}/rooms/${ROOM_ID}/slots/list?date=${DATE}"
