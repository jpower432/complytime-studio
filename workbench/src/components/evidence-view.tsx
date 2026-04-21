// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect, useRef } from "preact/hooks";
import { apiFetch } from "../api/fetch";

interface EvidenceRecord {
  evidence_id: string;
  policy_id: string;
  target_id: string;
  target_name?: string;
  target_type?: string;
  target_env?: string;
  control_id: string;
  rule_id: string;
  eval_result: string;
  engine_version?: string;
  frameworks?: string[];
  requirements?: string[];
  owner?: string;
  collected_at: string;
}

export function EvidenceView() {
  const [records, setRecords] = useState<EvidenceRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [policyId, setPolicyId] = useState("");
  const [controlId, setControlId] = useState("");
  const [targetName, setTargetName] = useState("");
  const [targetType, setTargetType] = useState("");
  const [targetEnv, setTargetEnv] = useState("");
  const [framework, setFramework] = useState("");
  const [engineVersion, setEngineVersion] = useState("");
  const [owner, setOwner] = useState("");
  const [start, setStart] = useState("");
  const [end, setEnd] = useState("");
  const [uploadStatus, setUploadStatus] = useState("");
  const fileRef = useRef<HTMLInputElement>(null);

  const search = () => {
    setLoading(true);
    const params = new URLSearchParams();
    if (policyId) params.set("policy_id", policyId);
    if (controlId) params.set("control_id", controlId);
    if (targetName) params.set("target_name", targetName);
    if (targetType) params.set("target_type", targetType);
    if (targetEnv) params.set("target_env", targetEnv);
    if (framework) params.set("framework", framework);
    if (engineVersion) params.set("engine_version", engineVersion);
    if (owner) params.set("owner", owner);
    if (start) params.set("start", start);
    if (end) params.set("end", end);
    params.set("limit", "200");

    apiFetch(`/api/evidence?${params}`)
      .then((r) => r.json())
      .then(setRecords)
      .catch(() => setRecords([]))
      .finally(() => setLoading(false));
  };

  useEffect(search, []);

  const handleUpload = async () => {
    const file = fileRef.current?.files?.[0];
    if (!file) return;
    setUploadStatus("Uploading...");
    const formData = new FormData();
    formData.append("file", file);
    try {
      const res = await apiFetch("/api/evidence/upload", { method: "POST", body: formData });
      const data = await res.json();
      setUploadStatus(`Imported ${data.inserted} rows, ${data.failed} failed`);
      search();
    } catch (e) {
      setUploadStatus(`Upload failed: ${e}`);
    }
  };

  return (
    <div class="evidence-view">
      <h2>Evidence</h2>

      <div class="evidence-filters">
        <input placeholder="Policy ID" value={policyId} onInput={(e) => setPolicyId((e.target as HTMLInputElement).value)} />
        <input placeholder="Control ID" value={controlId} onInput={(e) => setControlId((e.target as HTMLInputElement).value)} />
        <input placeholder="Framework" value={framework} onInput={(e) => setFramework((e.target as HTMLInputElement).value)} />
        <input placeholder="Target name" value={targetName} onInput={(e) => setTargetName((e.target as HTMLInputElement).value)} />
        <input placeholder="Target type" value={targetType} onInput={(e) => setTargetType((e.target as HTMLInputElement).value)} />
        <input placeholder="Environment" value={targetEnv} onInput={(e) => setTargetEnv((e.target as HTMLInputElement).value)} />
        <input placeholder="Version" value={engineVersion} onInput={(e) => setEngineVersion((e.target as HTMLInputElement).value)} />
        <input placeholder="Owner" value={owner} onInput={(e) => setOwner((e.target as HTMLInputElement).value)} />
        <input type="date" value={start} onInput={(e) => setStart((e.target as HTMLInputElement).value)} />
        <input type="date" value={end} onInput={(e) => setEnd((e.target as HTMLInputElement).value)} />
        <button class="btn btn-primary" onClick={search}>Search</button>
      </div>

      <div class="evidence-upload">
        <input type="file" ref={fileRef} accept=".csv,.json" />
        <button class="btn btn-secondary" onClick={handleUpload}>Upload</button>
        {uploadStatus && <span class="upload-status">{uploadStatus}</span>}
      </div>

      {loading ? (
        <div class="view-loading">Querying evidence...</div>
      ) : records.length === 0 ? (
        <div class="empty-state">
          <p>No evidence found. Adjust filters or upload evidence files.</p>
        </div>
      ) : (
        <table class="data-table">
          <thead>
            <tr>
              <th>Target</th>
              <th>Type</th>
              <th>Env</th>
              <th>Control</th>
              <th>Framework</th>
              <th>Result</th>
              <th>Collected</th>
            </tr>
          </thead>
          <tbody>
            {records.map((r) => (
              <tr key={r.evidence_id}>
                <td title={r.target_id}>{r.target_name || r.target_id}</td>
                <td>{r.target_type || "—"}</td>
                <td>{r.target_env || "—"}</td>
                <td>{r.control_id}</td>
                <td>{r.frameworks?.join(", ") || "—"}</td>
                <td><span class={`eval-badge eval-${r.eval_result?.toLowerCase().replace(/ /g, "-")}`}>{r.eval_result}</span></td>
                <td>{new Date(r.collected_at).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
