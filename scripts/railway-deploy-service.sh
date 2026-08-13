#!/usr/bin/env bash

set -euo pipefail

service_name="${1:?usage: railway-deploy-service.sh <service-name>}"
project_id="${RAILWAY_PROJECT_ID:?RAILWAY_PROJECT_ID is required}"
environment_name="${RAILWAY_ENVIRONMENT_NAME:-production}"

if [[ -n "${RAILWAY_CONFIG_B64:-}" ]]; then
  mkdir -p "$HOME/.railway"
  printf '%s' "$RAILWAY_CONFIG_B64" | base64 --decode >"$HOME/.railway/config.json"
  chmod 600 "$HOME/.railway/config.json"
fi

latest_deployment() {
  railway deployment list \
    --project "$project_id" \
    --environment "$environment_name" \
    --service "$service_name" \
    --limit 1 \
    --json
}

previous_id="$(latest_deployment | jq -r '.[0].id // empty')"

railway redeploy \
  --project "$project_id" \
  --environment "$environment_name" \
  --service "$service_name" \
  --from-source \
  --yes \
  --json >/dev/null

for _ in $(seq 1 180); do
  deployment_json="$(latest_deployment)"
  deployment_id="$(jq -r '.[0].id // empty' <<<"$deployment_json")"
  deployment_status="$(jq -r '.[0].status // empty' <<<"$deployment_json")"

  if [[ -z "$deployment_id" || "$deployment_id" == "$previous_id" ]]; then
    sleep 5
    continue
  fi

  printf '%s: %s\n' "$service_name" "$deployment_status"
  case "$deployment_status" in
    SUCCESS)
      exit 0
      ;;
    FAILED|CRASHED|REMOVED|SKIPPED|CANCELED|CANCELLED)
      railway logs \
        --project "$project_id" \
        --environment "$environment_name" \
        --service "$service_name" \
        --latest \
        --lines 100 || true
      exit 1
      ;;
  esac

  sleep 5
done

printf 'Timed out waiting for %s to deploy.\n' "$service_name" >&2
exit 1
