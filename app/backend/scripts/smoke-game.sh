#!/usr/bin/env sh
set -eu

api_url="${API_URL:-http://localhost:8080}"
email="smoke-$(date +%s)-$$@example.com"
password="avitosha-smoke-password"

require_command() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "required command is missing: $1" >&2
    exit 1
  }
}

require_command curl
require_command jq
require_command uuidgen

register_response="$(curl --fail --silent --show-error \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$email\",\"password\":\"$password\"}" \
  "$api_url/api/auth/register")"
access_token="$(printf '%s' "$register_response" | jq -er '.access_token')"
auth_header="Authorization: Bearer $access_token"

curl --fail --silent --show-error -H "$auth_header" "$api_url/api/v1/pet" | jq -e '.mood == "CALM"' >/dev/null
task_id="$(curl --fail --silent --show-error -H "$auth_header" "$api_url/api/v1/tasks" | jq -er '.tasks[0].id')"

index=1
while [ "$index" -le 5 ]; do
  event_id="$(uuidgen | tr '[:upper:]' '[:lower:]')"
  curl --fail --silent --show-error \
    -H "$auth_header" \
    -H 'Content-Type: application/json' \
    -d "{\"eventId\":\"$event_id\",\"type\":\"AD_VIEWED\",\"entityId\":\"smoke-ad-$index\",\"category\":\"FURNITURE\",\"occurredAt\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",\"metadata\":{\"source\":\"smoke\"}}" \
    "$api_url/api/v1/actions" | jq -e '.duplicate == false' >/dev/null
  index=$((index + 1))
done

curl --fail --silent --show-error -H "$auth_header" "$api_url/api/v1/tasks/$task_id" | jq -e '.status == "REWARDED" and .progress == 5' >/dev/null
curl --fail --silent --show-error -H "$auth_header" "$api_url/api/v1/pet" | jq -e '.growthXp == 30 and .mood == "PROUD"' >/dev/null
curl --fail --silent --show-error -H "$auth_header" "$api_url/api/v1/room" | jq -e '.items[] | select(.code == "DESK" and .status == "PLACED")' >/dev/null
curl --fail --silent --show-error -H "$auth_header" "$api_url/api/v1/story" | jq -e '.currentStage == 1' >/dev/null
curl --fail --silent --show-error -H "$auth_header" "$api_url/api/v1/daily-summary" | jq -e '.actionsCount == 5 and .earnedXp == 30' >/dev/null
curl --fail --silent --show-error -H "$auth_header" "$api_url/api/v1/leaderboard?period=weekly&limit=10" | jq -e '.currentUser.score == 100' >/dev/null
curl --fail --silent --show-error -H "$auth_header" "$api_url/api/v1/rewards/balance" | jq -e '.balances[] | select(.type == "AVITO_BONUS" and .balance == 12 and .earnedTotal == 12)' >/dev/null

echo "Avitosha smoke scenario passed for $email"
