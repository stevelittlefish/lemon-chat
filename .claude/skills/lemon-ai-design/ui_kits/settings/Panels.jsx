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

function AccountPanel({ s, set }) {
  const [showPwd, setShowPwd] = React.useState(false);
  const [pwdMode, setPwdMode] = React.useState(false); // expand the password panel

  const initials = (s.name || "?")
    .split(" ").map(w => w[0]).slice(0, 2).join("").toUpperCase();

  return (
    <>
      <Section title="Profile" lead="How you show up in chats and shared conversations.">
        <div className="profile-row">
          <div className="avatar-block">
            <div className="avatar-lg" style={{ background: s.avatarColor }}>
              {initials}
            </div>
            <button className="btn btn-secondary btn-sm">
              <Icon name="camera" size={14} />
              Change
            </button>
          </div>
          <div className="profile-fields">
            <Field label="Display name" desc="Shown above your messages.">
              <input
                className="input"
                style={{ width: 220 }}
                value={s.name}
                onChange={(e) => set("name", e.target.value)}
              />
            </Field>
            <Field label="Email" desc="For password reset only. We don't email you.">
              <input
                className="input"
                style={{ width: 220, fontFamily: "var(--font-mono)", fontSize: 13 }}
                type="email"
                value={s.email}
                onChange={(e) => set("email", e.target.value)}
              />
            </Field>
            <Field label="Username" desc="Used in shared chat URLs.">
              <div className="input-prefix">
                <span className="prefix">lemon.ai/@</span>
                <input
                  className="input"
                  style={{ width: 140, fontFamily: "var(--font-mono)", fontSize: 13 }}
                  value={s.username}
                  onChange={(e) => set("username", e.target.value)}
                />
              </div>
            </Field>
          </div>
        </div>
      </Section>

      <Section title="Password" lead="Used when syncing across devices. Stays on this machine otherwise.">
        {!pwdMode ? (
          <Field label="Password" desc={`Last changed ${s.passwordChanged}.`}>
            <button className="btn btn-secondary btn-sm" onClick={() => setPwdMode(true)}>
              <Icon name="lock" size={14} />
              Change password
            </button>
          </Field>
        ) : (
          <div className="pwd-form">
            <div className="pwd-row">
              <label>Current password</label>
              <div className="pwd-input-wrap">
                <input
                  type={showPwd ? "text" : "password"}
                  className="input"
                  placeholder="••••••••"
                />
                <button
                  className="pwd-toggle"
                  onClick={() => setShowPwd(!showPwd)}
                  title={showPwd ? "Hide" : "Show"}
                >
                  <Icon name="eye" size={14} />
                </button>
              </div>
            </div>
            <div className="pwd-row">
              <label>New password</label>
              <div className="pwd-input-wrap">
                <input
                  type={showPwd ? "text" : "password"}
                  className="input"
                  placeholder="at least 12 characters"
                />
              </div>
            </div>
            <div className="pwd-row">
              <label>Confirm new password</label>
              <div className="pwd-input-wrap">
                <input
                  type={showPwd ? "text" : "password"}
                  className="input"
                  placeholder="type it again"
                />
              </div>
            </div>
            <div className="pwd-strength">
              <span className="strength-bar"><span className="strength-fill medium"></span></span>
              <span className="strength-label">Decent. A long passphrase is better than a clever short one.</span>
            </div>
            <div className="pwd-actions">
              <button className="btn btn-ghost btn-sm" onClick={() => setPwdMode(false)}>Cancel</button>
              <button className="btn btn-primary btn-sm">Update password</button>
            </div>
          </div>
        )}
        <Field label="Two-factor auth" desc="Authenticator app or hardware key. Strongly recommended.">
          <Toggle value={s.twoFactor} onChange={(v) => set("twoFactor", v)} />
        </Field>
        <Field label="Active sessions" desc="3 devices signed in.">
          <button className="btn btn-secondary btn-sm">Review</button>
        </Field>
      </Section>

      <Section title="Danger zone" lead="">
        <div className="danger-row">
          <div className="meta">
            <span className="lbl">Sign out everywhere</span>
            <span className="desc">Ends every session except this one.</span>
          </div>
          <button className="btn btn-secondary btn-sm">Sign out all</button>
        </div>
        <div className="danger-row">
          <div className="meta">
            <span className="lbl">Delete account</span>
            <span className="desc">Erases your profile and synced chats. Local files stay on this machine.</span>
          </div>
          <button className="btn btn-danger btn-sm">Delete…</button>
        </div>
      </Section>
    </>
  );
}

function NotificationsPanel({ s, set }) {
  return (
    <>
      <Section title="Desktop" lead="Quiet by default. Lemon AI doesn't bug you while you're thinking.">
        <Field label="Show notifications" desc="When the app is in the background.">
          <Toggle value={s.notifyDesktop} onChange={(v) => set("notifyDesktop", v)} />
        </Field>
        <Field label="Play a sound" desc="A soft chime when a long response finishes.">
          <Toggle value={s.notifySound} onChange={(v) => set("notifySound", v)} />
        </Field>
        <Field label="Notify after" desc="Only ping if a generation takes longer than this.">
          <Select
            value={s.notifyThreshold}
            onChange={(v) => set("notifyThreshold", v)}
            options={[
              { value: "5",   label: "5 seconds" },
              { value: "15",  label: "15 seconds" },
              { value: "30",  label: "30 seconds" },
              { value: "60",  label: "1 minute" },
              { value: "off", label: "Never" },
            ]}
          />
        </Field>
      </Section>

      <Section title="Email" lead="Only the things we think you'd actually want.">
        <Field label="Security alerts" desc="New sign-ins, password changes. Always on.">
          <Toggle value={true} onChange={() => {}} />
        </Field>
        <Field label="Product updates" desc="A short note when we ship something good. Roughly monthly.">
          <Toggle value={s.emailUpdates} onChange={(v) => set("emailUpdates", v)} />
        </Field>
        <Field label="Community digest" desc="Cool things people built with Lemon AI. Weekly.">
          <Toggle value={s.emailDigest} onChange={(v) => set("emailDigest", v)} />
        </Field>
      </Section>

      <Section title="Quiet hours" lead="">
        <Field label="Mute notifications" desc="No desktop alerts during these hours.">
          <Toggle value={s.quietHours} onChange={(v) => set("quietHours", v)} />
        </Field>
        {s.quietHours && (
          <Field label="Hours" desc="Local time.">
            <div className="time-range">
              <input className="input input-sm" type="time" value={s.quietFrom} onChange={(e) => set("quietFrom", e.target.value)} />
              <span className="dash">→</span>
              <input className="input input-sm" type="time" value={s.quietTo} onChange={(e) => set("quietTo", e.target.value)} />
            </div>
          </Field>
        )}
      </Section>
    </>
  );
}

Object.assign(window, { ModelsPanel, AppearancePanel, PrivacyPanel, AdvancedPanel, AboutPanel, AccountPanel, NotificationsPanel });
