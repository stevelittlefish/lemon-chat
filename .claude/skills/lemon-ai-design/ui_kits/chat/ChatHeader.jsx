// ChatHeader.jsx — top bar: editable title + model picker

function ChatHeader({ title, onTitleChange, model, onModelChange }) {
  const [editing, setEditing] = React.useState(false);
  const inputRef = React.useRef();

  React.useEffect(() => {
    if (editing) inputRef.current?.select();
  }, [editing]);

  return (
    <header className="chat-header">
      {editing ? (
        <input
          ref={inputRef}
          className="ti ti-edit"
          defaultValue={title}
          onBlur={(e) => { onTitleChange(e.target.value || "Untitled"); setEditing(false); }}
          onKeyDown={(e) => {
            if (e.key === "Enter") e.target.blur();
            if (e.key === "Escape") { e.target.value = title; e.target.blur(); }
          }}
        />
      ) : (
        <div className="ti ti-edit" onClick={() => setEditing(true)}>{title}</div>
      )}
      <ModelPicker model={model} onChange={onModelChange} />
    </header>
  );
}

window.ChatHeader = ChatHeader;
