#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

# 1. healthz
curl -s "$BASE_URL/healthz" | grep -q ok
echo "OK: healthz"

# 2. dodavanje poruke (status 201)
STATUS=$(curl -s -o /dev/null -w '%{http_code}' -X POST -H "Content-Type: application/json" -d '{"author":"smoke","text":"hello"}' "$BASE_URL/api/messages")
[ "$STATUS" -eq 201 ]
echo "OK: dodavanje poruke"

# 3. autor smoke u listi poruka
curl -s "$BASE_URL/api/messages" | jq -r '.[].author' | grep -q smoke
echo "OK: autor smoke na listi"

# 4. prazna poruka vraća status 400
STATUS=$(curl -s -o /dev/null -w '%{http_code}' -X POST -H "Content-Type: application/json" -d '{"author":"smoke","text":""}' "$BASE_URL/api/messages")
[ "$STATUS" -eq 400 ]
echo "OK: prazna poruka vraca 400"

# 5. brisanje najnovije poruke (204) i GET (404)
ID=$(curl -s "$BASE_URL/api/messages" | jq -r '.[0].id')
DEL_STATUS=$(curl -s -o /dev/null -w '%{http_code}' -X DELETE "$BASE_URL/api/messages/$ID")
[ "$DEL_STATUS" -eq 204 ]

GET_STATUS=$(curl -s -o /dev/null -w '%{http_code}' "$BASE_URL/api/messages/$ID")
[ "$GET_STATUS" -eq 404 ]
echo "OK: brisanje i GET 404"
