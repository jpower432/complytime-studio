// SPDX-License-Identifier: Apache-2.0
import { useEffect, useRef } from "preact/hooks";
import type { EditorView } from "codemirror";

interface YamlEditorProps { content: string; onChange?: (value: string) => void }

export function YamlEditor({ content, onChange }: YamlEditorProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const contentRef = useRef(content);
  useEffect(() => {
    if (!containerRef.current) return;
    let view: EditorView;
    const init = async () => {
      const { EditorView: EV, basicSetup } = await import("codemirror");
      const { yaml } = await import("@codemirror/lang-yaml");
      const { oneDark } = await import("@codemirror/theme-one-dark");
      const { search } = await import("@codemirror/search");
      const { EditorState } = await import("@codemirror/state");
      const updateListener = EV.updateListener.of((update) => { if (update.docChanged) { const val = update.state.doc.toString(); contentRef.current = val; onChange?.(val); } });
      view = new EV({ state: EditorState.create({ doc: content, extensions: [basicSetup, yaml(), oneDark, search(), updateListener] }), parent: containerRef.current! });
      viewRef.current = view;
    };
    init().catch(() => { if (containerRef.current) containerRef.current.innerHTML = `<pre class="editor-fallback">${content.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}</pre>`; });
    return () => { view?.destroy(); viewRef.current = null; };
  }, []);
  useEffect(() => {
    const view = viewRef.current;
    if (view && content !== contentRef.current) { contentRef.current = content; view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: content } }); }
  }, [content]);
  return <div class="artifact-editor" ref={containerRef} />;
}
