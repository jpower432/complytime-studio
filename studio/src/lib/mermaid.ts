// SPDX-License-Identifier: Apache-2.0

const CDN_URL = "https://cdn.jsdelivr.net/npm/mermaid@latest/dist/mermaid.esm.min.mjs";

let mermaidMod: any = null;
let loading: Promise<any> | null = null;

async function loadMermaid(): Promise<any> {
  if (mermaidMod) return mermaidMod;
  if (!loading) {
    loading = import(/* @vite-ignore */ CDN_URL).then((mod) => {
      mermaidMod = mod.default;
      mermaidMod.initialize({ startOnLoad: false, theme: "neutral" });
      return mermaidMod;
    });
  }
  return loading;
}

export async function renderMermaidBlocks(container: HTMLElement): Promise<void> {
  const nodes = container.querySelectorAll<HTMLElement>("div.mermaid:not([data-processed])");
  if (nodes.length === 0) return;

  const mm = await loadMermaid();
  await mm.run({ nodes: Array.from(nodes) });
}
