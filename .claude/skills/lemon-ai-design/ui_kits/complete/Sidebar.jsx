// Sidebar.jsx — left rail for the Complete surface.
//   brand · local badge
//   Chat / Complete segmented switcher  ← the navigation entry point
//   New completion
//   Saved / recent completions list
//   Settings link

function Sidebar({ saved, activeId, onSelect, onNew, chatHref }) {
  return (
    <aside className="cmp-sidebar">
      <div className="cmp-sidebar-head">
        <div className="brand">
          <img src="../../assets/logo-mark.svg" alt="" />
          lemon ai
        </div>
        <span className="running" title="A model is loaded and ready">local</span>
      </div>

      <nav className="mode-switch" aria-label="Workspace">
        <a href={chatHref}>
          <Icon name="message" size={15} /> Chat
        </a>
        <a href="#" className="active" aria-current="page" onClick={(e) => e.preventDefault()}>
          <Icon name="pencil" size={15} /> Complete
        </a>
      </nav>

      <button className="new-cmp" onClick={onNew}>
        <Icon name="plus" size={16} />
        New completion
      </button>

      <div className="saved-label">Saved completions</div>
      <div className="saved-list">
        {saved.map((s) => (
          <div
            key={s.id}
            className={"saved-row" + (s.id === activeId ? " active" : "")}
            onClick={() => onSelect(s.id)}
          >
            <span className="st">{s.title}</span>
            <span className="ss">{s.meta}</span>
          </div>
        ))}
      </div>

      <div className="cmp-sidebar-foot">
        <a href="../settings/index.html"><Icon name="settings" size={15} /> Settings</a>
      </div>
    </aside>
  );
}

window.Sidebar = Sidebar;
