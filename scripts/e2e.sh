#!/usr/bin/env sh
set -eu

api_url="${WASATEXT_API_URL:-http://127.0.0.1:3000}"
frontend_url="${WASATEXT_FRONTEND_URL:-http://127.0.0.1:8080}"
username="e2e_$(date +%s)"
password="e2e-password-2026"
cookie_jar="$(mktemp)"
headers="$(mktemp)"
response="$(mktemp)"
trap 'rm -f "$cookie_jar" "$headers" "$response"' EXIT

register_status="$(curl -sS -c "$cookie_jar" -D "$headers" -o "$response" -w '%{http_code}' \
  -H 'Origin: http://localhost:8080' -H 'Content-Type: application/json' \
  --data "{\"name\":\"$username\",\"password\":\"$password\"}" "$api_url/users")"
test "$register_status" = "201"
grep -qi 'set-cookie: wasatext_session=.*HttpOnly.*SameSite=Lax' "$headers"

list_status="$(curl -sS -b "$cookie_jar" -o "$response" -w '%{http_code}' "$api_url/conversations?limit=10&offset=0")"
test "$list_status" = "200"

logout_status="$(curl -sS -b "$cookie_jar" -c "$cookie_jar" -o "$response" -w '%{http_code}' -X DELETE "$api_url/session")"
test "$logout_status" = "204"

revoked_status="$(curl -sS -b "$cookie_jar" -o "$response" -w '%{http_code}' "$api_url/conversations")"
test "$revoked_status" = "401"

frontend_status="$(curl -sS -D "$headers" -o "$response" -w '%{http_code}' "$frontend_url/")"
test "$frontend_status" = "200"
grep -qi 'content-security-policy:' "$headers"

printf 'E2E passed: register=%s auth=%s logout=%s revoked=%s frontend=%s\n' \
  "$register_status" "$list_status" "$logout_status" "$revoked_status" "$frontend_status"
