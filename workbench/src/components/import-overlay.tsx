// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect, useRef, useCallback } from "preact/hooks";
import { apiFetch } from "../api/fetch";
import { detectGemaraMetadataType } from "../lib/detect-gemara-type";

const ACCEPT = ".yaml,.yml,.json,application/json,text/yaml,text/x-yaml";

export interface ImportOverlayProps {
  open: boolean;
  onClose: () => void;
  expectedArtifactType?: string;
  onSuccess?: () => void;
}

export function ImportOverlay({ open, onClose, expectedArtifactType, onSuccess }: ImportOverlayProps) {
  const [dragActive, setDragActive] = useState(false);
  const [fileName, setFileName] = useState("");
  const [rawText, setRawText] = useState<string | null>(null);
  const [detectedType, setDetectedType] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState<{ ok: boolean; message: string } | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const resetFileState = useCallback(() => {
    setFileName("");
    setRawText(null);
    setDetectedType(null);
    setResult(null);
    if (inputRef.current) {
      inputRef.current.value = "";
    }
  }, []);

  useEffect(() => {
    if (!open) {
      resetFileState();
      setSubmitting(false);
    }
  }, [open, resetFileState]);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        onClose();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  const processFile = (file: File) => {
    const lower = file.name.toLowerCase();
    const ok =
      lower.endsWith(".yaml") || lower.endsWith(".yml") || lower.endsWith(".json");
    if (!ok) {
      setResult({ ok: false, message: "Choose a .yaml, .yml, or .json file." });
      return;
    }
    setResult(null);
    const reader = new FileReader();
    reader.onload = () => {
      const text = typeof reader.result === "string" ? reader.result : "";
      setFileName(file.name);
      setRawText(text);
      setDetectedType(detectGemaraMetadataType(text));
    };
    reader.onerror = () => {
      setResult({ ok: false, message: "Could not read file." });
    };
    reader.readAsText(file, "UTF-8");
  };

  const contentTypeForFile = (name: string): string =>
    name.toLowerCase().endsWith(".json") ? "application/json" : "text/yaml";

  const doImport = async () => {
    if (!rawText?.trim()) return;
    setSubmitting(true);
    setResult(null);
    try {
      const res = await apiFetch("/api/import", {
        method: "POST",
        headers: { "Content-Type": contentTypeForFile(fileName || "") },
        body: rawText,
      });
      const txt = await res.text();
      if (!res.ok) {
        setResult({ ok: false, message: txt || `Import failed (${res.status})` });
        return;
      }
      let msg = "Import succeeded.";
      try {
        const j = JSON.parse(txt) as Record<string, string>;
        if (j.policy_id) msg = `Imported policy ${j.policy_id}.`;
        else if (j.catalog_id) msg = `Imported ${j.catalog_type || "catalog"} ${j.catalog_id}.`;
        else if (j.mapping_id) msg = `Imported mapping ${j.mapping_id}.`;
        else if (j.status) msg = j.status;
      } catch {
        if (txt) msg = txt;
      }
      setResult({ ok: true, message: msg });
      onSuccess?.();
    } catch (e) {
      setResult({ ok: false, message: String(e) });
    } finally {
      setSubmitting(false);
    }
  };

  const typeMismatch =
    expectedArtifactType &&
    detectedType &&
    detectedType !== expectedArtifactType;

  if (!open) {
    return null;
  }

  return (
    <div
      class="import-overlay"
      role="presentation"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div class="import-modal" role="dialog" aria-modal="true" aria-labelledby="import-title">
        <h3 id="import-title">Import artifact</h3>
        {expectedArtifactType && (
          <p style={{ margin: "0 0 12px", fontSize: "13px", color: "var(--text-muted)" }}>
            Expected: { expectedArtifactType }
          </p>
        )}
        <label
          class={`import-dropzone ${dragActive ? "drag-active" : ""}`}
          onDragEnter={(e) => {
            e.preventDefault();
            setDragActive(true);
          }}
          onDragOver={(e) => {
            e.preventDefault();
            setDragActive(true);
          }}
          onDragLeave={() => setDragActive(false)}
          onDrop={(e) => {
            e.preventDefault();
            setDragActive(false);
            const f = e.dataTransfer?.files?.[0];
            if (f) processFile(f);
          }}
        >
          <input
            ref={inputRef}
            type="file"
            accept={ACCEPT}
            onChange={(e) => {
              const f = (e.target as HTMLInputElement).files?.[0];
              if (f) processFile(f);
            }}
          />
          Drop a Gemara artifact or click to browse
        </label>
        {(fileName || detectedType) && (
          <div class="import-file-info">
            {fileName && <span>{fileName}</span>}
            {detectedType && (
              <span class="import-type-badge">{detectedType}</span>
            )}
          </div>
        )}
        {typeMismatch && (
          <div class="import-result import-result-error" style={{ marginTop: "12px" }}>
            File type is {detectedType}, not {expectedArtifactType}. You can still import.
          </div>
        )}
        <div class="import-actions">
          <button type="button" class="btn btn-secondary" onClick={onClose} disabled={submitting}>
            Cancel
          </button>
          <button
            type="button"
            class="btn btn-primary"
            disabled={submitting || !rawText?.trim()}
            onClick={() => void doImport()}
          >
            {submitting ? "Importing…" : "Import"}
          </button>
        </div>
        {result && (
          <div
            class={`import-result ${
              result.ok ? "import-result-success" : "import-result-error"
            }`}
            role={result.ok ? "status" : "alert"}
          >
            {result.message}
          </div>
        )}
      </div>
    </div>
  );
}
