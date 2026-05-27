// SettingsNav.jsx — left rail with sections.

const SECTIONS = [
  { id: "models",     label: "Models",     icon: "sparkle" },
  { id: "appearance", label: "Appearance", icon: "panel" },
  { id: "privacy",    label: "Privacy",    icon: "check" },
  { id: "advanced",   label: "Advanced",   icon: "settings" },
  { id: "about",      label: "About",      icon: "message" },
];

function SettingsNav({ activeId, onSelect }) {
  return (
    <nav className="snav">
      <div className="snav-head">
        <img src="../../assets/logo-mark.svg" alt="" />
        <div className="title">Settings</div>
        <div className="sub">v0.4.1</div>
      </div>
      {SECTIONS.map(s => (
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
