// ParamsPanel.jsx — generation parameters, rendered two ways:
//   ParamsRail — vertical right-hand rail (layout-rail)
//   ParamsBar  — horizontal strip under the editor (layout-bar)
// Both drive the same lifted state from App.

// Stop sequences are literal strings. Since the field is single-line, users
// type C-style escapes (\n, \t, \r) which we expand into real characters;
// chips render them escaped again so they stay readable.
const expandEscapes = (s) =>
  s.replace(/\\n/g, "\n").replace(/\\t/g, "\t").replace(/\\r/g, "\r");
const showEscapes = (s) =>
  s.replace(/\n/g, "\\n").replace(/\t/g, "\\t").replace(/\r/g, "\\r");

function StopSequences({ stops, onAdd, onRemove, hint }) {
  const [draft, setDraft] = React.useState("");
  const commit = () => {
    const v = expandEscapes(draft.trim());
    if (v) onAdd(v);
    setDraft("");
  };
  return (
    <div className="stop-wrap">
      <div className="stop-chips">
        {stops.map((s, i) => (
          <span className="stop-chip" key={s + i}>
            <code>{showEscapes(s)}</code>
            <button onClick={() => onRemove(i)} aria-label={"Remove " + showEscapes(s)}>
              <Icon name="x" size={11} />
            </button>
          </span>
        ))}
        <input
          className="stop-add"
          placeholder="+ add"
          title={"Type a string to stop at. Use \\n for a newline, \\t for a tab. Enter to add."}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); commit(); } }}
          onBlur={commit}
        />
      </div>
      {hint && (
        <div className="stop-hint">
          Generation halts at any of these. Type <code>\n</code> for a newline.
        </div>
      )}
    </div>
  );
}

function ParamsRail({ model, onModel, temp, onTemp, maxTokens, onMaxTokens, stops, onAddStop, onRemoveStop }) {
  return (
    <aside className="cmp-rail">
      <div className="rail-head">
        <span className="ico"><Icon name="sliders" size={16} /></span>
        Parameters
      </div>
      <div className="rail-body">
        <div className="rail-modelwrap">
          <div className="lab">Model</div>
          <ModelPicker model={model} onChange={onModel} />
        </div>

        <div className="param-field">
          <div className="lab">
            <span>Temperature</span>
            <span className="val">{temp.toFixed(2)}</span>
          </div>
          <input
            className="param-range" type="range"
            min="0" max="2" step="0.05"
            value={temp}
            onChange={(e) => onTemp(parseFloat(e.target.value))}
          />
          <div style={{ display: "flex", justifyContent: "space-between", fontFamily: "var(--font-mono)", fontSize: 10, color: "var(--fg-faint)" }}>
            <span>precise</span><span>creative</span>
          </div>
        </div>

        <div className="param-field">
          <div className="lab">
            <span>Max tokens</span>
          </div>
          <input
            className="param-num" type="number"
            min="1" max="8192" step="1"
            value={maxTokens}
            onChange={(e) => onMaxTokens(Math.max(1, parseInt(e.target.value || "1", 10)))}
          />
        </div>

        <div className="param-field">
          <div className="lab"><span>Stop sequences</span></div>
          <StopSequences stops={stops} onAdd={onAddStop} onRemove={onRemoveStop} hint />
        </div>
      </div>
    </aside>
  );
}

function ParamsBar({ model, onModel, temp, onTemp, maxTokens, onMaxTokens, stops, onAddStop, onRemoveStop, children }) {
  return (
    <div className="param-bar">
      <div className="pb-item">
        <ModelPicker model={model} onChange={onModel} />
      </div>
      <div className="pb-divider"></div>
      <div className="pb-item">
        <span className="pb-lab">Temp</span>
        <input
          className="param-range pb-temp" type="range"
          min="0" max="2" step="0.05"
          value={temp}
          onChange={(e) => onTemp(parseFloat(e.target.value))}
        />
        <span className="pb-val">{temp.toFixed(2)}</span>
      </div>
      <div className="pb-divider"></div>
      <div className="pb-item">
        <span className="pb-lab">Max</span>
        <input
          className="pb-tokens" type="number"
          min="1" max="8192" step="1"
          value={maxTokens}
          onChange={(e) => onMaxTokens(Math.max(1, parseInt(e.target.value || "1", 10)))}
        />
      </div>
      <div className="pb-divider"></div>
      <div className="pb-item pb-stops">
        <span className="pb-lab">Stop</span>
        <StopSequences stops={stops} onAdd={onAddStop} onRemove={onRemoveStop} />
      </div>
      <div className="grow"></div>
      {children}
    </div>
  );
}

window.ParamsRail = ParamsRail;
window.ParamsBar = ParamsBar;
