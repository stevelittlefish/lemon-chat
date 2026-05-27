// App.jsx — root of settings UI kit.

const TITLES = {
  account:       ["Account",       "Your profile, password, and the keys to the kingdom."],
  notifications: ["Notifications", "When and how Lemon AI taps you on the shoulder."],
  models:        ["Models",        "Pick a default and tune how it generates."],
  appearance:    ["Appearance",    "Paper, ink, and how much they take up."],
  privacy:       ["Privacy",       "Everything off by default. Always."],
  advanced:      ["Advanced",      "For people who want to fiddle."],
  about:         ["About",         ""],
};

function App() {
  const [active, setActive] = React.useState("account");
  const [state, setState] = React.useState({
    // account
    name: "Wren Calloway",
    email: "wren@calloway.studio",
    username: "wren",
    avatarColor: "var(--lemon-300)",
    passwordChanged: "3 weeks ago",
    twoFactor: false,
    // notifications
    notifyDesktop: true,
    notifySound: false,
    notifyThreshold: "30",
    emailUpdates: true,
    emailDigest: false,
    quietHours: true,
    quietFrom: "22:00",
    quietTo: "08:00",
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
    case "account":       panel = <AccountPanel       s={state} set={set} />; break;
    case "notifications": panel = <NotificationsPanel s={state} set={set} />; break;
    case "models":        panel = <ModelsPanel        s={state} set={set} />; break;
    case "appearance":    panel = <AppearancePanel    s={state} set={set} />; break;
    case "privacy":       panel = <PrivacyPanel       s={state} set={set} />; break;
    case "advanced":      panel = <AdvancedPanel      s={state} set={set} />; break;
    case "about":         panel = <AboutPanel />; break;
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
