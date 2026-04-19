// SPDX-License-Identifier: Apache-2.0

import { useState, useRef, useEffect } from "preact/hooks";
import {
  allArtifacts,
  activeArtifactName,
  activateArtifact,
  removeArtifact,
  addArtifact,
  renameArtifact,
} from "../store/workspace";

function nextUntitledName(existing: string[]): string {
  let n = 1;
  while (existing.includes(`untitled-${n}.yaml`)) n++;
  return `untitled-${n}.yaml`;
}

function TabName({ name, onRename }: { name: string; onRename: (n: string) => void }) {
  const [editing, setEditing] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (editing && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }
  }, [editing]);

  function commit() {
    let val = inputRef.current?.value.trim();
    if (!val) { setEditing(false); return; }
    val = val.replace(/\.ya?ml$/i, "") + ".yaml";
    if (val !== name) onRename(val);
    setEditing(false);
  }

  if (editing) {
    return (
      <input
        ref={inputRef}
        class="workspace-tab-rename"
        defaultValue={name.replace(/\.ya?ml$/i, "")}
        onBlur={commit}
        onKeyDown={(e) => {
          if (e.key === "Enter") commit();
          if (e.key === "Escape") setEditing(false);
        }}
        onClick={(e) => e.stopPropagation()}
      />
    );
  }

  return (
    <span
      class="workspace-tab-name"
      onDblClick={(e) => { e.stopPropagation(); setEditing(true); }}
    >
      {name}
    </span>
  );
}

export function ArtifactTabs() {
  const artifacts = allArtifacts.value;
  const activeName = activeArtifactName.value;

  return (
    <div class="workspace-tabs">
      {artifacts.map((a) => (
        <div
          key={a.name}
          class={`workspace-tab ${a.name === activeName ? "active" : ""}`}
          onClick={() => activateArtifact(a.name)}
        >
          <TabName name={a.name} onRename={(n) => renameArtifact(a.name, n)} />
          <button
            class="workspace-tab-close"
            title="Close"
            onClick={(e) => {
              e.stopPropagation();
              removeArtifact(a.name);
            }}
          >
            &times;
          </button>
        </div>
      ))}
      <button
        class="workspace-tab workspace-tab-add"
        title="New artifact"
        onClick={() => addArtifact(nextUntitledName(artifacts.map((a) => a.name)), "")}
      >
        +
      </button>
    </div>
  );
}
