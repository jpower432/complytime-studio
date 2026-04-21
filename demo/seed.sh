#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Seed demo data into a running ComplyTime Studio instance.
# Usage: GATEWAY_URL=http://localhost:8080 ./demo/seed.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
STUDIO_API_TOKEN="${STUDIO_API_TOKEN:-}"

AUTH_HEADER=()
if [[ -n "${STUDIO_API_TOKEN}" ]]; then
  AUTH_HEADER=(-H "Authorization: Bearer ${STUDIO_API_TOKEN}")
fi

info()  { echo "==> $*"; }
check() { echo "  ✓ $*"; }

info "Seeding demo data into ${GATEWAY_URL}"

info "Importing policy..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  -X POST "${GATEWAY_URL}/api/policies/import" \
  -H "Content-Type: application/json" \
  "${AUTH_HEADER[@]}" \
  -d @"${SCRIPT_DIR}/policy.json")
if [[ "$HTTP_CODE" =~ ^2 ]]; then
  check "Policy imported (${HTTP_CODE})"
else
  echo "  ! Policy import returned ${HTTP_CODE} (may already exist)"
fi

info "Ingesting evidence (20 records across 2 targets)..."
RESULT=$(curl -s -X POST "${GATEWAY_URL}/api/evidence" \
  -H "Content-Type: application/json" \
  "${AUTH_HEADER[@]}" \
  -d @"${SCRIPT_DIR}/evidence.json")
echo "  ${RESULT}"

info "Verifying seed data..."

POLICY_COUNT=$(curl -s "${AUTH_HEADER[@]}" "${GATEWAY_URL}/api/policies" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "?")
check "Policies: ${POLICY_COUNT}"

EVIDENCE_COUNT=$(curl -s "${AUTH_HEADER[@]}" "${GATEWAY_URL}/api/evidence?policy_id=demo-cloud-native-security&limit=100" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "?")
check "Evidence records: ${EVIDENCE_COUNT}"

info ""
info "Demo data seeded. Open ${GATEWAY_URL} and try the prompts in demo/prompts.md"
