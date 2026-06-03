// ModelPicker.jsx — locally-installed models dropdown.

const MODELS = [
  { id: "llama-3.1-8b",     name: "llama-3.1-8b-instruct", note: "Solid all-rounder. Fast on most laptops.",     size: "4.9 GB", tps: 36, status: "loaded" },
  { id: "qwen-coder-7b",    name: "qwen2.5-coder-7b",      note: "Best for code. Reads diffs cleanly.",          size: "5.4 GB", tps: 28, status: "ready" },
  { id: "mistral-7b",       name: "mistral-7b-instruct",   note: "Quick replies. Light on memory.",              size: "4.1 GB", tps: 42, status: "ready" },
  { id: "phi-3.5-mini",     name: "phi-3.5-mini",          note: "Tiny. Good on older hardware.",                size: "2.2 GB", tps: 58, status: "ready" },
];

function ModelPicker({ model, onChange }) {
  const [open, setOpen] = React.useState(false);
  const ref = React.useRef();

  React.useEffect(() => {
    if (!open) return;
    const close = (e) => { if (!ref.current?.contains(e.target)) setOpen(false); };
    document.addEventListener("click", close);
    return () => document.removeEventListener("click", close);
  }, [open]);

  const current = MODELS.find(m => m.id === model.id) || MODELS[0];

  return (
    <div className="model-picker" ref={ref}>
      <button className="trigger" onClick={() => setOpen(o => !o)}>
        <img src="../../assets/sparkle.svg" width="14" height="14" style={{color: "var(--lemon-700)"}} alt="" />
        <span className="name">{current.name}</span>
        <span style={{ color: "var(--fg-faint)" }}>·</span>
        <span>{current.tps} tok/s</span>
        <Icon name="chevDown" size={14} style={{ color: "var(--fg-muted)" }} />
      </button>

      {open && (
        <div className="panel">
          <div style={{ padding: "6px 10px 8px", fontSize: 11, fontFamily: "var(--font-mono)", color: "var(--fg-faint)", textTransform: "uppercase", letterSpacing: "0.08em" }}>
            Installed
          </div>
          {MODELS.map(m => (
            <div
              key={m.id}
              className={"opt" + (m.id === current.id ? " sel" : "")}
              onClick={() => { onChange(m); setOpen(false); }}
            >
              <div className="check">
                {m.id === current.id && <Icon name="check" size={14} />}
              </div>
              <div className="body">
                <div className="nm">{m.name}</div>
                <div className="me">{m.note}</div>
                <div className="me" style={{ marginTop: 4, fontFamily: "var(--font-mono)", fontSize: 11 }}>
                  {m.size} · {m.tps} tok/s
                </div>
              </div>
            </div>
          ))}
          <div className="sep"></div>
          <div className="foot">
            <Icon name="download" size={13} />
            <a>Browse the model library →</a>
          </div>
        </div>
      )}
    </div>
  );
}

window.ModelPicker = ModelPicker;
window.MODELS = MODELS;
