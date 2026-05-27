// Field.jsx + Section.jsx + small controls.

function Section({ title, lead, children }) {
  return (
    <section className="section">
      <h2>{title}</h2>
      {lead && <p className="lead">{lead}</p>}
      {children}
    </section>
  );
}

function Field({ label, desc, children }) {
  return (
    <div className="field">
      <div className="meta">
        <span className="lbl">{label}</span>
        {desc && <span className="desc">{desc}</span>}
      </div>
      <div className="ctl">{children}</div>
    </div>
  );
}

function Toggle({ value, onChange }) {
  return (
    <div
      className={"toggle" + (value ? " on" : "")}
      onClick={() => onChange(!value)}
      role="switch"
      aria-checked={value}
    />
  );
}

function Segmented({ options, value, onChange }) {
  return (
    <div className="seg">
      {options.map(o => (
        <button
          key={o.value}
          className={value === o.value ? "on" : ""}
          onClick={() => onChange(o.value)}
        >
          {o.label}
        </button>
      ))}
    </div>
  );
}

function Slider({ value, onChange, min = 0, max = 1, step = 0.05 }) {
  return (
    <div className="slider-row">
      <input
        type="range"
        className="slider"
        min={min}
        max={max}
        step={step}
        value={value}
        onChange={(e) => onChange(parseFloat(e.target.value))}
      />
      <span className="value">{Number(value).toFixed(2)}</span>
    </div>
  );
}

function Stepper({ value, onChange, min = 0, max = 99 }) {
  const clamp = (v) => Math.min(max, Math.max(min, v));
  return (
    <div className="stepper">
      <button onClick={() => onChange(clamp(value - 1))}>−</button>
      <div className="val">{value}</div>
      <button onClick={() => onChange(clamp(value + 1))}>+</button>
    </div>
  );
}

function Select({ value, onChange, options }) {
  return (
    <select className="select" value={value} onChange={(e) => onChange(e.target.value)}>
      {options.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
    </select>
  );
}

Object.assign(window, { Section, Field, Toggle, Segmented, Slider, Stepper, Select });
