#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# DEMO CONTENT — synthetic data for local development and demos only.
# Not derived from real assessments. Do not use in production.
#
# Seed demo data into a running ComplyTime Studio instance.
# Usage: GATEWAY_URL=http://localhost:9090 STUDIO_API_TOKEN=studio-dev-token ./demo/seed.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATEWAY_URL="${GATEWAY_URL:-http://localhost:9090}"
STUDIO_API_TOKEN="${STUDIO_API_TOKEN:-}"

AUTH_HEADER=()
if [[ -n "${STUDIO_API_TOKEN}" ]]; then
  AUTH_HEADER=(-H "Authorization: Bearer ${STUDIO_API_TOKEN}")
fi

info()  { echo "==> $*"; }
check() { echo "  ✓ $*"; }
warn()  { echo "  ! $*"; }

post_file() {
  local endpoint="$1" file="$2" label="$3"
  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "${GATEWAY_URL}${endpoint}" \
    -H "Content-Type: application/json" \
    "${AUTH_HEADER[@]}" \
    -d @"${file}")
  if [[ "$HTTP_CODE" =~ ^2 ]]; then
    check "${label} (${HTTP_CODE})"
  else
    warn "${label} returned ${HTTP_CODE} (may already exist)"
  fi
}

info "Seeding demo data into ${GATEWAY_URL}"

# ── Policies ──
info "Importing policies..."
post_file "/api/policies/import" "${SCRIPT_DIR}/policy.json" "ampel-branch-protection"
post_file "/api/policies/import" "${SCRIPT_DIR}/policy-kube-security.json" "kube-security-baseline"
post_file "/api/policies/import" "${SCRIPT_DIR}/policy-supply-chain.json" "supply-chain-attestation"

# ── Evidence (Gemara EvaluationLog artifacts) ──
info "Ingesting evidence..."
for artifact in "${SCRIPT_DIR}"/eval-*.yaml; do
  [ -f "${artifact}" ] || continue
  name="$(basename "${artifact}")"
  HTTP_CODE=$(curl -s -o /tmp/seed_ingest_resp -w "%{http_code}" \
    -X POST "${GATEWAY_URL}/api/evidence/ingest" \
    -H "Content-Type: application/x-yaml" \
    "${AUTH_HEADER[@]}" \
    --data-binary @"${artifact}")
  if [[ "$HTTP_CODE" =~ ^2 ]]; then
    inserted=$(python3 -c "import sys,json; print(json.load(sys.stdin).get('inserted','?'))" < /tmp/seed_ingest_resp 2>/dev/null || echo "?")
    check "${name}: ${inserted} rows (${HTTP_CODE})"
  else
    warn "${name} returned ${HTTP_CODE}"
  fi
done

# ── Verification ──
info "Verifying seed data..."

POLICY_COUNT=$(curl -s "${AUTH_HEADER[@]}" "${GATEWAY_URL}/api/policies" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "?")
check "Policies: ${POLICY_COUNT} (expected 3)"

EVIDENCE_COUNT=$(curl -s "${AUTH_HEADER[@]}" "${GATEWAY_URL}/api/evidence?limit=500" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "?")
check "Evidence records: ${EVIDENCE_COUNT}"

info ""
info "Demo data seeded. Open the gateway URL and explore the dashboard."
