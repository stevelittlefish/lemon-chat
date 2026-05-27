// App.jsx — root of settings UI kit.

const TITLES = {
  models:     ["Models",     "Pick a default and tune how it generates."],
  appearance: ["Appearance", "Paper, ink, and how much they take up."],
  privacy:    ["Privacy",    "Everything off by default. Always."],
  advanced:   ["Advanced",   "For people who want to fiddle."],
  about:      ["About",      ""],
};

function App() {
  const [active, setActive] = React.useState("models");
  const [state, setState] = React.useState({
    // models
    defaultModel: "llama-3.1-8b",
    temperature: 0.7,
    topP: 0.9,
    maxContext: "8192",
    systemPrompt: "You are a helpful assistant. Be direct. Use plain words.",
    // appearance
    theme: "system",
    density: "comfortable",
    fontSize: 15,
    showTimestamps: false,
    groupByDay: true,
    // privacy
    saveLocal: true,
    crashReports: false,
    analytics: false,
    // advanced
    serverUrl: "http://localhost:11434",
    autoStart: true,
    gpuLayers: 33,
    beta: false,
    showTokens: false,
  });

  const set = (k, v) => setState(prev => ({ ...prev, [k]: v }));

  const [title, desc] = TITLES[active];

  let panel;
  switch (active) {
    case "models":     panel = <ModelsPanel     s={state} set={set} />; break;
    case "appearance": panel = <AppearancePanel s={state} set={set} />; break;
    case "privacy":    panel = <PrivacyPanel    s={state} set={set} />; break;
    case "advanced":   panel = <AdvancedPanel   s={state} set={set} />; break;
    case "about":      panel = <AboutPanel />; break;
  }

  return (
    <div className="app">
      <SettingsNav activeId={active} onSelect={setActive} />
      <main className="smain">
        <div className="smain-head">
          <h1>{title}</h1>
          {desc && <p>{desc}</p>}
        </div>
        {panel}
      </main>
    </div>
  );
}

window.App = App;
