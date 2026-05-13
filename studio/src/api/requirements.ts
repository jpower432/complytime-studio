// SPDX-License-Identifier: Apache-2.0

import { apiFetch } from "./fetch";

export interface RequirementRow {
  catalog_id: string;
  control_id: string;
  control_title: string;
  requirement_id: string;
  requirement_text: string;
  evidence_count: number;
  latest_evidence?: string;
  classification?: string;
}

export interface RequirementEvidenceRow {
  evidence_id: string;
  target_id: string;
  target_name?: string;
  rule_id: string;
  eval_result: string;
  classification?: string;
  assessed_at?: string;
  collected_at: string;
  source_registry?: string;
}

export interface RequirementMatrixParams {
  policy_id: string;
  audit_start?: string;
  audit_end?: string;
  classification?: string;
  control_family?: string;
  limit?: number;
  offset?: number;
}

export async function fetchRequirementMatrix(
  params: RequirementMatrixParams,
): Promise<RequirementRow[]> {
  const qs = new URLSearchParams();
  qs.set("policy_id", params.policy_id);
  if (params.audit_start) qs.set("audit_start", params.audit_start);
  if (params.audit_end) qs.set("audit_end", params.audit_end);
  if (params.classification) qs.set("classification", params.classification);
  if (params.control_family) qs.set("control_family", params.control_family);
  if (params.limit) qs.set("limit", String(params.limit));
  if (params.offset) qs.set("offset", String(params.offset));

  const res = await apiFetch(`/api/requirements?${qs}`);
  if (!res.ok) throw new Error(`requirements: ${res.status}`);
  return res.json();
}

export async function fetchRequirementEvidence(
  requirementId: string,
  params: RequirementMatrixParams,
): Promise<RequirementEvidenceRow[]> {
  const qs = new URLSearchParams();
  qs.set("policy_id", params.policy_id);
  if (params.audit_start) qs.set("audit_start", params.audit_start);
  if (params.audit_end) qs.set("audit_end", params.audit_end);
  if (params.limit) qs.set("limit", String(params.limit));
  if (params.offset) qs.set("offset", String(params.offset));

  const res = await apiFetch(`/api/requirements/${encodeURIComponent(requirementId)}/evidence?${qs}`);
  if (!res.ok) throw new Error(`requirement evidence: ${res.status}`);
  return res.json();
}
