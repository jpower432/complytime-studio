#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# DEMO CONTENT — synthetic data for local development and demos only.
# Not derived from real assessments. Do not use in production.
#
# Seed demo data into a running ComplyTime Studio instance.
#
# When OAuth is enabled, port-forward directly to the gateway container
# (bypassing the OAuth2 Proxy sidecar) for unauthenticated seeding:
#   kubectl port-forward -n complytime deployment/studio-gateway 9090:8080
#   GATEWAY_URL=http://localhost:9090 ./demo/seed.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATEWAY_URL="${GATEWAY_URL:-http://localhost:9090}"
SEED_IDENTITY="${SEED_IDENTITY:-seed@complytime.dev}"

AUTH_HEADER=(-H "X-Forwarded-Email: ${SEED_IDENTITY}")

info()  { echo "==> $*"; }
check() { echo "  ✓ $*"; }
warn()  { echo "  ! $*"; }

ingest_file() {
  local file="$1" label="$2" ct="${3:-application/x-yaml}"
  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "${GATEWAY_URL}/api/ingest" \
    -H "Content-Type: ${ct}" \
    "${AUTH_HEADER[@]}" \
    --data-binary @"${file}")
  if [[ "$HTTP_CODE" =~ ^2 ]]; then
    check "${label} (${HTTP_CODE})"
  else
    warn "${label} returned ${HTTP_CODE} (may already exist)"
  fi
}

info "Seeding demo data into ${GATEWAY_URL}"

# ── Policies ──
# Seed files store policies as JSON wrappers; the `content` field holds the
# Gemara Policy YAML that POST /api/ingest expects.
info "Importing policies..."
ingest_policy() {
  local file="$1" label="$2"
  local yaml_content
  yaml_content=$(python3 -c "import sys,json; print(json.load(open(sys.argv[1]))['content'])" "${file}")
  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "${GATEWAY_URL}/api/ingest" \
    -H "Content-Type: application/x-yaml" \
    "${AUTH_HEADER[@]}" \
    --data-binary "${yaml_content}")
  if [[ "$HTTP_CODE" =~ ^2 ]]; then
    check "${label} (${HTTP_CODE})"
  else
    warn "${label} returned ${HTTP_CODE} (may already exist)"
  fi
}
ingest_policy "${SCRIPT_DIR}/policy.json" "ampel-branch-protection"
ingest_policy "${SCRIPT_DIR}/policy-kube-security.json" "kube-security-baseline"
ingest_policy "${SCRIPT_DIR}/policy-supply-chain.json" "supply-chain-attestation"

# ── Evidence (Gemara EvaluationLog artifacts) ──
info "Ingesting evidence..."
for artifact in "${SCRIPT_DIR}"/eval-*.yaml; do
  [ -f "${artifact}" ] || continue
  name="$(basename "${artifact}")"
  ingest_file "${artifact}" "${name}"
done

# ── Verification ──
info "Verifying seed data..."

POLICY_COUNT=$(curl -s "${AUTH_HEADER[@]}" "${GATEWAY_URL}/api/policies" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "?")
check "Policies: ${POLICY_COUNT} (expected 3)"

EVIDENCE_COUNT=$(curl -s "${AUTH_HEADER[@]}" "${GATEWAY_URL}/api/evidence?limit=500" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "?")
check "Evidence records: ${EVIDENCE_COUNT}"

info ""
info "Demo data seeded. Open the gateway URL and explore the dashboard."
