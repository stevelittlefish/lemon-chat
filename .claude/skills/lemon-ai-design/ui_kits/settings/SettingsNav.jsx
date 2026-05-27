// SettingsNav.jsx — left rail with sections.

const SECTIONS = [
  { id: "account",       label: "Account",       icon: "user",     group: "you" },
  { id: "notifications", label: "Notifications", icon: "bell",     group: "you" },
  { id: "models",        label: "Models",        icon: "sparkle",  group: "app" },
  { id: "appearance",    label: "Appearance",    icon: "panel",    group: "app" },
  { id: "privacy",       label: "Privacy",       icon: "lock",     group: "app" },
  { id: "advanced",      label: "Advanced",      icon: "settings", group: "app" },
  { id: "about",         label: "About",         icon: "message",  group: "app" },
];

function SettingsNav({ activeId, onSelect }) {
  const youItems = SECTIONS.filter(s => s.group === "you");
  const appItems = SECTIONS.filter(s => s.group === "app");
  return (
    <nav className="snav">
      <a href="../chat/index.html" className="snav-back" title="Back to chat">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"
             strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M19 12H5 M12 19l-7-7 7-7" />
        </svg>
        Back to chat
      </a>
      <div className="snav-head">
        <img src="../../assets/logo-mark.svg" alt="" />
        <div className="title">Settings</div>
        <div className="sub">v0.4.1</div>
      </div>
      <div className="snav-group-label">You</div>
      {youItems.map(s => (
        <div
          key={s.id}
          className={"snav-item" + (s.id === activeId ? " active" : "")}
          onClick={() => onSelect(s.id)}
        >
          <Icon name={s.icon} size={16} />
          {s.label}
        </div>
      ))}
      <div className="snav-group-label">App</div>
      {appItems.map(s => (
        <div
          key={s.id}
          className={"snav-item" + (s.id === activeId ? " active" : "")}
          onClick={() => onSelect(s.id)}
        >
          <Icon name={s.icon} size={16} />
          {s.label}
        </div>
      ))}
    </nav>
  );
}

window.SettingsNav = SettingsNav;
window.SETTINGS_SECTIONS = SECTIONS;
