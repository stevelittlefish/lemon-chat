// Panels — each section's contents.

const INSTALLED_MODELS = [
  { id: "llama-3.1-8b", name: "llama-3.1-8b-instruct", size: "4.9 GB", quant: "Q4_K_M", desc: "Solid all-rounder. Fast on most laptops.", default: true,  loaded: true },
  { id: "qwen-coder-7b", name: "qwen2.5-coder-7b",     size: "5.4 GB", quant: "Q5_K_M", desc: "Best for code. Reads diffs cleanly.",      default: false, loaded: false },
  { id: "mistral-7b",    name: "mistral-7b-instruct",  size: "4.1 GB", quant: "Q4_K_M", desc: "Quick replies. Light on memory.",          default: false, loaded: false },
  { id: "phi-3.5-mini",  name: "phi-3.5-mini",         size: "2.2 GB", quant: "Q5_K_M", desc: "Tiny. Good on older hardware.",            default: false, loaded: false },
];

function ModelsPanel({ s, set }) {
  return (
    <>
      <Section title="Installed models" lead="Pick a default. The chat will load it when you open the app.">
        <div className="model-list">
          {INSTALLED_MODELS.map(m => (
            <div key={m.id} className={"model-row" + (m.id === s.defaultModel ? " default" : "")}>
              <div className="info">
                <div className="nm">
                  {m.name}
                  {m.loaded && <span className="badge live"><span className="dot"></span> loaded</span>}
                  {m.id === s.defaultModel && <span className="badge accent"><span className="dot"></span> default</span>}
                </div>
                <div className="desc">{m.desc}</div>
                <div className="me">{m.size} · {m.quant}</div>
              </div>
              <button className="btn btn-secondary btn-sm" onClick={() => set("defaultModel", m.id)}
                disabled={m.id === s.defaultModel}>
                {m.id === s.defaultModel ? "Default" : "Set default"}
              </button>
              <button className="menubtn" title="More"><Icon name="more" size={16} /></button>
            </div>
          ))}
        </div>
        <div className="download-card">
          <Icon name="download" size={22} style={{ color: "var(--fg-muted)" }} />
          <div className="desc">
            <div className="title">Add a model</div>
            <div className="sub">Browse the library, or paste a Hugging Face URL.</div>
          </div>
          <button className="btn btn-primary btn-sm">Browse library</button>
        </div>
      </Section>

      <Section title="Generation" lead="Defaults applied to every new chat. You can override per-chat from the model picker.">
        <Field label="Temperature" desc="Lower is more focused. Higher is more varied.">
          <Slider value={s.temperature} onChange={(v) => set("temperature", v)} min={0} max={1.5} step={0.05} />
        </Field>
        <Field label="Top-p" desc="Cuts off unlikely tokens. 1.0 keeps all of them.">
          <Slider value={s.topP} onChange={(v) => set("topP", v)} min={0.1} max={1} step={0.01} />
        </Field>
        <Field label="Max context" desc="Tokens of conversation kept in memory.">
          <Select
            value={s.maxContext}
            onChange={(v) => set("maxContext", v)}
            options={[
              { value: "4096",  label: "4,096 tokens" },
              { value: "8192",  label: "8,192 tokens" },
              { value: "16384", label: "16,384 tokens" },
              { value: "32768", label: "32,768 tokens" },
            ]}
          />
        </Field>
        <Field label="System prompt" desc="Sent at the start of every conversation.">
        </Field>
        <textarea
          className="prompt"
          placeholder="You are a helpful assistant…"
          value={s.systemPrompt}
          onChange={(e) => set("systemPrompt", e.target.value)}
        />
      </Section>
    </>
  );
}

function AppearancePanel({ s, set }) {
  return (
    <Section title="Appearance" lead="Light, dark, or whatever your OS is doing.">
      <Field label="Theme" desc="We default to whatever your system is set to.">
        <Segmented
          value={s.theme}
          onChange={(v) => set("theme", v)}
          options={[
            { value: "system", label: "Match system" },
            { value: "light",  label: "Light" },
            { value: "dark",   label: "Dark" },
          ]}
        />
      </Field>
      <Field label="Density" desc="How much breathing room between messages.">
        <Segmented
          value={s.density}
          onChange={(v) => set("density", v)}
          options={[
            { value: "comfortable", label: "Comfortable" },
            { value: "compact",     label: "Compact" },
          ]}
        />
      </Field>
      <Field label="Font size" desc="Applies to chat messages.">
        <Stepper value={s.fontSize} onChange={(v) => set("fontSize", v)} min={12} max={22} />
      </Field>
      <Field label="Show timestamps" desc="Next to each message in a chat.">
        <Toggle value={s.showTimestamps} onChange={(v) => set("showTimestamps", v)} />
      </Field>
      <Field label="Sidebar grouping" desc="Group conversations by day.">
        <Toggle value={s.groupByDay} onChange={(v) => set("groupByDay", v)} />
      </Field>
    </Section>
  );
}

function PrivacyPanel({ s, set }) {
  return (
    <Section title="Privacy" lead="Nothing leaves this machine unless you turn something on here. We don't have an account system.">
      <Field label="Save conversations locally" desc="Stored in ~/Library/Application Support/lemon-ai. Off means each session starts blank.">
        <Toggle value={s.saveLocal} onChange={(v) => set("saveLocal", v)} />
      </Field>
      <Field label="Anonymous crash reports" desc="If we crash, send the stack trace. No conversation contents, ever.">
        <Toggle value={s.crashReports} onChange={(v) => set("crashReports", v)} />
      </Field>
      <Field label="Analytics" desc="Off by default. We genuinely don't want them.">
        <Toggle value={s.analytics} onChange={(v) => set("analytics", v)} />
      </Field>
      <Field label="Clear all conversations" desc="Cannot be undone.">
        <button className="btn btn-danger btn-sm">Clear all…</button>
      </Field>
    </Section>
  );
}

function AdvancedPanel({ s, set }) {
  return (
    <>
      <Section title="Server" lead="Lemon AI talks to a local inference server. Change this if you're running your own.">
        <Field label="Server URL" desc="Where the model server is listening.">
          <input
            className="input"
            style={{ width: 220, fontFamily: "var(--font-mono)", fontSize: 13 }}
            value={s.serverUrl}
            onChange={(e) => set("serverUrl", e.target.value)}
          />
        </Field>
        <Field label="Auto-start on launch" desc="Boot the server when the app opens.">
          <Toggle value={s.autoStart} onChange={(v) => set("autoStart", v)} />
        </Field>
        <Field label="GPU offload" desc="Number of model layers pushed to the GPU. Higher is faster, more memory.">
          <Stepper value={s.gpuLayers} onChange={(v) => set("gpuLayers", v)} min={0} max={99} />
        </Field>
      </Section>

      <Section title="Experimental" lead="Things we're still figuring out. Expect rough edges.">
        <Field label="Beta features" desc="Show in-progress features in the menu.">
          <Toggle value={s.beta} onChange={(v) => set("beta", v)} />
        </Field>
        <Field label="Show token usage" desc="Per-message prompt + completion token counts.">
          <Toggle value={s.showTokens} onChange={(v) => set("showTokens", v)} />
        </Field>
      </Section>
    </>
  );
}

function AboutPanel() {
  return (
    <Section title="About" lead="">
      <div className="about">
        <img src="../../assets/logo-mark.svg" alt="" />
        <h2>lemon ai</h2>
        <div className="version">version 0.4.1 — Earl Grey</div>
        <p>A chat client for AI that runs on your own machine. Open source. Built by a small team that gets tired of subscriptions.</p>
        <div className="about-links">
          <a>github</a>
          <a>docs</a>
          <a>changelog</a>
          <a>discord</a>
        </div>
      </div>
    </Section>
  );
}

Object.assign(window, { ModelsPanel, AppearancePanel, PrivacyPanel, AdvancedPanel, AboutPanel });
