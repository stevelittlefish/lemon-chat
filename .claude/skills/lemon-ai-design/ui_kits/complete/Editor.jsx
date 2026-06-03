// Editor.jsx — the dual-mode writing surface.
//   write  → plain <textarea>, edits the whole document as raw text
//   read   → markdown + LaTeX rendered; the freshly-generated tail is
//            wrapped in .cmp-gen (a lemon wash that settles to normal)
// A blinking caret trails the generated text while streaming.

// Render a markdown string into a node, then typeset any LaTeX inside it.
function Rendered({ source, className }) {
  const ref = React.useRef(null);
  React.useEffect(() => {
    const el = ref.current;
    if (!el) return;
    let html = "";
    try { html = window.marked ? window.marked.parse(source || "") : (source || ""); }
    catch (e) { html = source || ""; }
    el.innerHTML = html;
    if (window.renderMathInElement) {
      try {
        window.renderMathInElement(el, {
          delimiters: [
            { left: "$$", right: "$$", display: true },
            { left: "$",  right: "$",  display: false },
            { left: "\\(", right: "\\)", display: false },
            { left: "\\[", right: "\\]", display: true },
          ],
          throwOnError: false,
        });
      } catch (e) {}
    }
  }, [source]);
  return <div ref={ref} className={className}></div>;
}

function Editor({
  text, mode, streaming, genStart, settled,
  onChange, taRef,
}) {
  const scrollRef = React.useRef(null);

  // Keep the view pinned to the bottom while tokens stream in.
  React.useEffect(() => {
    if (streaming && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [text, streaming]);

  const hasGen = genStart != null && genStart < text.length;
  const promptPart = hasGen ? text.slice(0, genStart) : text;
  const genPart    = hasGen ? text.slice(genStart) : "";

  return (
    <div className="cmp-editor-scroll" ref={scrollRef}>
      {mode === "write" ? (
        <div className="cmp-doc">
          <textarea
            ref={taRef}
            className="cmp-write"
            value={text}
            placeholder="Start writing, then press Run and the model continues from where you stop…"
            onChange={(e) => onChange(e.target.value)}
            spellCheck={false}
          />
        </div>
      ) : (
        <div className="cmp-doc">
          {promptPart && <Rendered source={promptPart} className="cmp-read" />}
          {hasGen && (
            <div className={"cmp-read cmp-gen" + (settled ? " settled" : "")}>
              <Rendered source={genPart} className="" />
              {streaming && <span className="cmp-caret"></span>}
            </div>
          )}
          {!promptPart && !hasGen && (
            <div className="cmp-read" style={{ color: "var(--fg-faint)" }}>
              Nothing here yet.
            </div>
          )}
        </div>
      )}
    </div>
  );
}

window.Editor = Editor;
