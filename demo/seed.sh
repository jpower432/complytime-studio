#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# DEMO CONTENT — synthetic data for local development and demos only.
# Not derived from real assessments. Do not use in production.
#
# Seed demo data into a running ComplyTime Studio instance.
# Usage:
#   GATEWAY_URL=http://localhost:9090 \
#   STUDIO_API_TOKEN=studio-dev-token \
#   REGISTRY_URL=localhost:5000 \
#   ./demo/seed.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATEWAY_URL="${GATEWAY_URL:-http://localhost:9090}"
STUDIO_API_TOKEN="${STUDIO_API_TOKEN:-}"
REGISTRY_URL="${REGISTRY_URL:-localhost:5000}"
REGISTRY_INTERNAL="${REGISTRY_INTERNAL:-studio-registry:5000}"

AUTH_HEADER=()
if [[ -n "${STUDIO_API_TOKEN}" ]]; then
  AUTH_HEADER=(-H "Authorization: Bearer ${STUDIO_API_TOKEN}")
fi

info()  { echo "==> $*"; }
check() { echo "  ✓ $*" >&2; }
warn()  { echo "  ! $*" >&2; }

push_and_import() {
  local yaml_file="$1" repo_path="$2" tag="$3" label="$4"

  oras push --plain-http --disable-path-validation \
    --artifact-type "application/vnd.gemara.bundle.v1" \
    "${REGISTRY_URL}/${repo_path}:${tag}" \
    "${yaml_file}:application/vnd.gemara.artifact.v1+yaml" \
    >/dev/null 2>&1

  local ref="${REGISTRY_INTERNAL}/${repo_path}:${tag}"
  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "${GATEWAY_URL}/api/import" \
    -H "Content-Type: application/json" \
    "${AUTH_HEADER[@]}" \
    -d "{\"reference\": \"${ref}\"}")
  if [[ "$HTTP_CODE" =~ ^2 ]]; then
    check "${label} (${HTTP_CODE})"
  else
    warn "${label} import returned ${HTTP_CODE}"
  fi
}

extract_yaml() {
  python3 -c "import json,sys; print(json.load(open(sys.argv[1]))['content'], end='')" "$1"
}

# POST raw YAML or JSON to unified import (metadata.type auto-detected).
post_import() {
  local file="$1" label="$2"
  local ctype="application/x-yaml"
  case "${file}" in
    *.json) ctype="application/json" ;;
    *.yaml|*.yml) ctype="application/x-yaml" ;;
  esac
  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "${GATEWAY_URL}/api/import" \
    -H "Content-Type: ${ctype}" \
    "${AUTH_HEADER[@]}" \
    --data-binary @"${file}")
  if [[ "$HTTP_CODE" =~ ^2 ]]; then
    check "${label} (${HTTP_CODE})"
  else
    warn "${label} returned ${HTTP_CODE} (may already exist)"
  fi
}

program_id_by_name() {
  local name="$1"
  local body
  body=$(curl -sf "${GATEWAY_URL}/api/programs" "${AUTH_HEADER[@]}") || return 0
  printf '%s' "${body}" | python3 -c "
import json, sys
want = sys.argv[1]
try:
    data = json.load(sys.stdin)
except json.JSONDecodeError:
    sys.exit(0)
for row in data:
    if row.get(\"name\") == want:
        print(row[\"id\"])
        break
" "${name}"
}

create_program_if_missing() {
  local name="$1"
  local json_body="$2"
  local existing pid
  existing="$(program_id_by_name "${name}")"
  if [[ -n "${existing}" ]]; then
    check "Program already exists: ${name} (${existing})"
    printf '%s' "${existing}"
    return 0
  fi
  HTTP_CODE=$(curl -s -o /tmp/seed_program_resp -w "%{http_code}" \
    -X POST "${GATEWAY_URL}/api/programs" \
    -H "Content-Type: application/json" \
    "${AUTH_HEADER[@]}" \
    -d "${json_body}")
  if [[ "${HTTP_CODE}" =~ ^2 ]]; then
    pid="$(python3 -c "import json; print(json.load(open('/tmp/seed_program_resp')).get('id',''))")"
    if [[ -n "${pid}" ]]; then
      check "Created program ${name} (${pid})"
      printf '%s' "${pid}"
    else
      warn "Created ${name} but response had no id (HTTP ${HTTP_CODE})"
      printf ''
    fi
  else
    warn "Failed to create ${name} (HTTP ${HTTP_CODE})"
    printf ''
  fi
}

if ! command -v oras &>/dev/null; then
  echo "oras CLI is required. Install: https://oras.land/docs/installation"
  exit 1
fi

info "Seeding demo data into ${GATEWAY_URL} (registry: ${REGISTRY_URL})"

# ── Policies ──
info "Importing policies..."
TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT

for pfile in "${SCRIPT_DIR}"/policy*.json; do
  [ -f "${pfile}" ] || continue
  name="$(basename "${pfile}" .json)"
  extract_yaml "${pfile}" > "${TMPDIR}/${name}.yaml"
  post_import "${TMPDIR}/${name}.yaml" "${name}"
done

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

# ── Guidance catalogs (unified import) ──
info "Importing guidance catalogs..."
for artifact in "${SCRIPT_DIR}"/guidance-*.yaml "${SCRIPT_DIR}"/guidance-*.yml; do
  [ -f "${artifact}" ] || continue
  stem="$(basename "${artifact}")"
  stem="${stem%.yaml}"
  stem="${stem%.yml}"
  post_import "${artifact}" "${stem}"
done

# ── Mapping documents ──
info "Importing mapping documents..."
for artifact in "${SCRIPT_DIR}"/mapping-*.json; do
  [ -f "${artifact}" ] || continue
  stem="$(basename "${artifact}" .json)"
  extract_yaml "${artifact}" > "${TMPDIR}/${stem}.yaml"
  post_import "${TMPDIR}/${stem}.yaml" "${stem}"
done

for artifact in "${SCRIPT_DIR}"/mapping-*.yaml "${SCRIPT_DIR}"/mapping-*.yml; do
  [ -f "${artifact}" ] || continue
  stem="$(basename "${artifact}")"
  stem="${stem%.yaml}"
  stem="${stem%.yml}"
  post_import "${artifact}" "${stem}"
done

# ── Programs ──
info "Creating programs..."
sleep 2

FEDRAMP_ID="$(
  create_program_if_missing "FedRAMP Moderate" '{
    "name": "FedRAMP Moderate",
    "framework": "FedRAMP",
    "applicability": ["moderate"],
    "description": "FedRAMP Moderate authorization program",
    "status": "active",
    "environments": ["production", "staging"]
  }'
)"

create_program_if_missing "Supply Chain Security" '{
  "name": "Supply Chain Security",
  "framework": "SLSA",
  "applicability": ["build", "source"],
  "description": "Software supply chain security program",
  "status": "intake",
  "environments": ["ci"]
}' >/dev/null

# ── Assign policies to programs ──
info "Assigning policies to programs..."
POLICIES_JSON="$(curl -sf "${GATEWAY_URL}/api/policies" "${AUTH_HEADER[@]}")" || POLICIES_JSON="[]"

if [[ -n "${FEDRAMP_ID}" ]]; then
  UPDATED="$(
    POLICIES_JSON="${POLICIES_JSON}" python3 <<'PY'
import json, os

policies = json.loads(os.environ.get("POLICIES_JSON") or "[]")
sel = [
    p["policy_id"]
    for p in policies
    if p.get("title") and ("branch" in p["title"].lower() or "kube" in p["title"].lower())
]
if not sel:
    print("")
else:
    print(json.dumps(sel))
PY
  )"
  if [[ -n "${UPDATED}" ]]; then
    CURRENT="$(curl -sf "${GATEWAY_URL}/api/programs/${FEDRAMP_ID}" "${AUTH_HEADER[@]}")" || CURRENT=""
    if [[ -n "${CURRENT}" ]]; then
      BODY="$(
        CURRENT_JSON="${CURRENT}" SEL_JSON="${UPDATED}" python3 <<'PY'
import json, os

cur = json.loads(os.environ["CURRENT_JSON"])
sel = json.loads(os.environ["SEL_JSON"])
cur["policy_ids"] = sel
print(json.dumps(cur))
PY
      )"
      HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        -X PUT "${GATEWAY_URL}/api/programs/${FEDRAMP_ID}" \
        -H "Content-Type: application/json" \
        "${AUTH_HEADER[@]}" \
        -d "${BODY}")
      if [[ "${HTTP_CODE}" =~ ^2 ]]; then
        check "Assigned branch/kube policies to FedRAMP Moderate (${HTTP_CODE})"
      else
        warn "FedRAMP policy assign returned ${HTTP_CODE} (may already match or version conflict)"
      fi
    else
      warn "Could not load program ${FEDRAMP_ID} for policy assign"
    fi
  fi
fi

# ── Verification ──
info "Verifying seed data..."

POLICY_COUNT=$(curl -s "${AUTH_HEADER[@]}" "${GATEWAY_URL}/api/policies" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "?")
check "Policies: ${POLICY_COUNT} (expected 3)"

EVIDENCE_COUNT=$(curl -s "${AUTH_HEADER[@]}" "${GATEWAY_URL}/api/evidence?limit=500" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "?")
check "Evidence records: ${EVIDENCE_COUNT}"

PROGRAM_COUNT=$(curl -s "${AUTH_HEADER[@]}" "${GATEWAY_URL}/api/programs" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "?")
check "Programs: ${PROGRAM_COUNT} (expected >= 2)"

info ""
info "Demo data seeded. Open the gateway URL and explore the dashboard."
