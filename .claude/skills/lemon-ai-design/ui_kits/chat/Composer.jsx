// Composer.jsx — input area + send + attach button.

function Composer({ onSend, disabled, initial }) {
  const [value, setValue] = React.useState("");
  const taRef = React.useRef();

  // When parent passes a new suggested prompt, drop it into the box.
  React.useEffect(() => {
    if (initial) {
      setValue(initial);
      // focus after the value lands
      setTimeout(() => taRef.current?.focus(), 0);
    }
  }, [initial]);

  React.useEffect(() => {
    const ta = taRef.current;
    if (!ta) return;
    ta.style.height = "auto";
    ta.style.height = Math.min(180, ta.scrollHeight) + "px";
  }, [value]);

  const send = () => {
    const v = value.trim();
    if (!v || disabled) return;
    onSend(v);
    setValue("");
  };

  return (
    <div className="composer-wrap">
      <div className="composer">
        <textarea
          ref={taRef}
          rows={1}
          placeholder="Ask anything…"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); send(); }
          }}
        />
        <div className="composer-acts">
          <button className="ghost-icon" title="Attach a file">
            <Icon name="attach" size={16} />
          </button>
          <button className="send-btn" onClick={send} disabled={!value.trim() || disabled} aria-label="Send">
            <Icon name="send" size={14} style={{ color: "var(--ink-900)" }} />
          </button>
        </div>
      </div>
      <div className="composer-hint">
        <kbd>Enter</kbd> to send · <kbd>Shift</kbd> + <kbd>Enter</kbd> for newline · runs on your machine
      </div>
    </div>
  );
}

window.Composer = Composer;
