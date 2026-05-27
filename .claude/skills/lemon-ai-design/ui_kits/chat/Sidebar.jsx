// Sidebar.jsx — left rail. Brand, new chat, search, conversation list, settings link.

function Sidebar({ chats, activeId, onSelect, onNew, onDelete, onRename }) {
  const [query, setQuery] = React.useState("");
  const [menuFor, setMenuFor] = React.useState(null);

  React.useEffect(() => {
    if (!menuFor) return;
    const close = (e) => {
      if (!e.target.closest(".row-menu") && !e.target.closest(".kebab")) setMenuFor(null);
    };
    document.addEventListener("click", close);
    return () => document.removeEventListener("click", close);
  }, [menuFor]);

  const filtered = chats.filter(c =>
    c.title.toLowerCase().includes(query.toLowerCase())
  );

  // Group by today / earlier
  const today = [], earlier = [];
  filtered.forEach(c => {
    (c.today ? today : earlier).push(c);
  });

  return (
    <aside className="sidebar">
      <div className="sidebar-head">
        <div className="brand">
          <img src="../../assets/logo-mark.svg" alt="" />
          lemon ai
        </div>
        <span className="running" title="A model is loaded and ready">local</span>
      </div>

      <button className="new-chat" onClick={onNew}>
        <Icon name="plus" size={16} />
        New chat
      </button>

      <div className="sidebar-search">
        <span className="ico"><Icon name="search" size={14} /></span>
        <input
          type="text"
          placeholder="Search chats…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
      </div>

      <div className="chats">
        {today.length > 0 && <div className="group-label">Today</div>}
        {today.map(c => (
          <ChatRow key={c.id} chat={c} active={c.id === activeId}
            onSelect={() => onSelect(c.id)}
            onMenuOpen={(e) => { e.stopPropagation(); setMenuFor(c.id); }}
            menuOpen={menuFor === c.id}
            onDelete={() => { onDelete(c.id); setMenuFor(null); }}
            onRename={() => { onRename(c.id); setMenuFor(null); }}
          />
        ))}
        {earlier.length > 0 && <div className="group-label">Earlier</div>}
        {earlier.map(c => (
          <ChatRow key={c.id} chat={c} active={c.id === activeId}
            onSelect={() => onSelect(c.id)}
            onMenuOpen={(e) => { e.stopPropagation(); setMenuFor(c.id); }}
            menuOpen={menuFor === c.id}
            onDelete={() => { onDelete(c.id); setMenuFor(null); }}
            onRename={() => { onRename(c.id); setMenuFor(null); }}
          />
        ))}
      </div>

      <div className="sidebar-foot">
        <a><Icon name="settings" size={15} /> Settings</a>
      </div>
    </aside>
  );
}

function ChatRow({ chat, active, onSelect, onMenuOpen, menuOpen, onDelete, onRename }) {
  return (
    <div className={"chat-row" + (active ? " active" : "")} onClick={onSelect}>
      <span className="title">{chat.title}</span>
      <div
        className="kebab"
        onClick={onMenuOpen}
        title="More"
      >
        <Icon name="more" size={14} />
      </div>
      {menuOpen && (
        <div className="menu row-menu" style={{
          position: "absolute",
          top: "calc(100% + 4px)",
          right: 8,
          zIndex: 30,
        }}>
          <div className="menu-item" onClick={(e) => { e.stopPropagation(); onRename(); }}>
            <Icon name="edit" size={14} /> Rename
          </div>
          <div className="menu-sep"></div>
          <div className="menu-item danger" onClick={(e) => { e.stopPropagation(); onDelete(); }}>
            <Icon name="trash" size={14} /> Delete chat
          </div>
        </div>
      )}
    </div>
  );
}

window.Sidebar = Sidebar;
