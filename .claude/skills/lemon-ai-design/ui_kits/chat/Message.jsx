// Message.jsx — a single message bubble (user or AI) + its action footer.
// Assistant: copy, regenerate (last only), thumbs, info (last only), delete.
// User:      edit, copy, delete.
// Info popover shows the metrics attached to the assistant message.

function Message({ msg, isLastAssistant, onDelete, onEdit, onRegenerate }) {
  const isUser = msg.role === "user";
  const [editing, setEditing] = React.useState(false);
  const [infoOpen, setInfoOpen] = React.useState(false);
  const [confirmOpen, setConfirmOpen] = React.useState(false);
  const [copied, setCopied] = React.useState(false);

  const copy = () => {
    navigator.clipboard?.writeText(msg.content).catch(() => {});
    setCopied(true);
    setTimeout(() => setCopied(false), 1200);
  };

  if (editing) {
    return (
      <MessageEditor
        initial={msg.content}
        isUser={isUser}
        onCancel={() => setEditing(false)}
        onSave={(t) => { setEditing(false); onEdit(t); }}
      />
    );
  }

  return (
    <div className={"msg " + (isUser ? "user" : "ai")}>
      <div className="meta">{isUser ? "you" : "lemon"}</div>
      <div className={"bubble " + (isUser ? "user" : "ai")}>
        {msg.streaming
          ? <TypingThenContent content={msg.content} />
          : <Rendered content={msg.content} />}
      </div>

      {!msg.streaming && (
        <div className={"msg-foot" + (isLastAssistant ? " pinned" : "")}>
          {isUser ? (
            <>
              <FootBtn title="Edit"    onClick={() => setEditing(true)} icon="edit" />
              <FootBtn title={copied ? "Copied" : "Copy"} onClick={copy} icon={copied ? "check" : "copy"} />
              <div className="info-wrap">
                <FootBtn title="Delete" onClick={() => setConfirmOpen(true)} icon="trash" danger active={confirmOpen} />
                {confirmOpen && (
                  <ConfirmPopover
                    label="Delete this message?"
                    onCancel={() => setConfirmOpen(false)}
                    onConfirm={() => { setConfirmOpen(false); onDelete(); }}
                    align="right"
                  />
                )}
              </div>
            </>
          ) : (
            <>
              <FootBtn title={copied ? "Copied" : "Copy"} onClick={copy} icon={copied ? "check" : "copy"} />
              {isLastAssistant && <FootBtn title="Regenerate" onClick={onRegenerate} icon="refresh" />}
              <FootBtn title="Good response" icon="thumbsUp" />
              <FootBtn title="Bad response" icon="thumbsDown" />
              {isLastAssistant && msg.metrics && (
                <div className="info-wrap">
                  <FootBtn
                    title="Run info"
                    onClick={() => setInfoOpen(v => !v)}
                    icon="info"
                    active={infoOpen}
                  />
                  {infoOpen && (
                    <InfoPopover
                      metrics={msg.metrics}
                      onClose={() => setInfoOpen(false)}
                    />
                  )}
                </div>
              )}
              <div className="info-wrap">
                <FootBtn title="Delete" onClick={() => setConfirmOpen(true)} icon="trash" danger active={confirmOpen} />
                {confirmOpen && (
                  <ConfirmPopover
                    label="Delete this message?"
                    onCancel={() => setConfirmOpen(false)}
                    onConfirm={() => { setConfirmOpen(false); onDelete(); }}
                  />
                )}
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
}

// ---------- small subcomponents ----------

function FootBtn({ title, onClick, icon, danger, active }) {
  let cls = "foot-btn";
  if (danger) cls += " danger";
  if (active) cls += " active";
  return (
    <button className={cls} title={title} aria-label={title} onClick={onClick}>
      <Icon name={icon} size={14} />
    </button>
  );
}

function MessageEditor({ initial, isUser, onSave, onCancel }) {
  const [val, setVal] = React.useState(initial);
  const taRef = React.useRef();

  React.useEffect(() => {
    const ta = taRef.current;
    if (!ta) return;
    ta.focus();
    ta.setSelectionRange(ta.value.length, ta.value.length);
    ta.style.height = "auto";
    ta.style.height = Math.min(360, ta.scrollHeight) + "px";
  }, []);

  const onChange = (e) => {
    setVal(e.target.value);
    const ta = e.target;
    ta.style.height = "auto";
    ta.style.height = Math.min(360, ta.scrollHeight) + "px";
  };

  const save = () => {
    const v = val.trim();
    if (!v) return;
    onSave(v);
  };

  return (
    <div className={"msg " + (isUser ? "user" : "ai")}>
      <div className="meta">{isUser ? "you · editing" : "lemon · editing"}</div>
      <div className={"msg-editor " + (isUser ? "user" : "ai")}>
        <textarea
          ref={taRef}
          value={val}
          onChange={onChange}
          onKeyDown={(e) => {
            if (e.key === "Escape") { e.preventDefault(); onCancel(); }
            if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) { e.preventDefault(); save(); }
          }}
        />
        <div className="msg-editor-acts">
          <span className="msg-editor-hint">
            <kbd>⌘</kbd>+<kbd>Enter</kbd> to save · <kbd>Esc</kbd> to cancel
            {isUser && <span> · sends a new response</span>}
          </span>
          <button className="btn-ghost-sm" onClick={onCancel}>Cancel</button>
          <button className="btn-primary-sm" onClick={save} disabled={!val.trim()}>
            {isUser ? "Save & rerun" : "Save"}
          </button>
        </div>
      </div>
    </div>
  );
}

function InfoPopover({ metrics, onClose }) {
  // close on outside click
  const ref = React.useRef();
  React.useEffect(() => {
    const onDoc = (e) => {
      if (ref.current && !ref.current.contains(e.target)) onClose();
    };
    // delay so the click that opened us doesn't immediately close it
    const t = setTimeout(() => document.addEventListener("mousedown", onDoc), 0);
    return () => { clearTimeout(t); document.removeEventListener("mousedown", onDoc); };
  }, [onClose]);

  const fmtMs = (ms) => ms >= 1000 ? (ms / 1000).toFixed(2) + " s" : ms + " ms";

  return (
    <div className="info-pop" ref={ref} role="dialog" aria-label="Run info">
      <div className="info-pop-head">
        <span className="info-pop-eyebrow">Run info</span>
        <button className="info-pop-x" onClick={onClose} aria-label="Close">
          <Icon name="close" size={12} />
        </button>
      </div>

      <dl className="info-pop-stats">
        <div>
          <dt>Prompt</dt>
          <dd>{metrics.promptTokens.toLocaleString()} <span>tok</span></dd>
        </div>
        <div>
          <dt>Response</dt>
          <dd>{metrics.responseTokens.toLocaleString()} <span>tok</span></dd>
        </div>
        <div>
          <dt>Total time</dt>
          <dd>{fmtMs(metrics.totalMs)}</dd>
        </div>
        <div>
          <dt>Throughput</dt>
          <dd>{metrics.tokensPerSec} <span>tok/s</span></dd>
        </div>
      </dl>

      <div className="info-pop-foot">
        <span>
          <Icon name="clock" size={10} /> first token in {fmtMs(metrics.ttftMs)}
        </span>
      </div>
    </div>
  );
}

function ConfirmPopover({ label, onCancel, onConfirm, align = "left" }) {
  const ref = React.useRef();
  React.useEffect(() => {
    const onDoc = (e) => {
      if (ref.current && !ref.current.contains(e.target)) onCancel();
    };
    const onKey = (e) => { if (e.key === "Escape") onCancel(); };
    const t = setTimeout(() => {
      document.addEventListener("mousedown", onDoc);
      document.addEventListener("keydown", onKey);
    }, 0);
    return () => {
      clearTimeout(t);
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [onCancel]);

  return (
    <div className={"confirm-pop " + align} ref={ref} role="dialog" aria-label={label}>
      <div className="confirm-pop-label">{label}</div>
      <div className="confirm-pop-acts">
        <button className="btn-ghost-sm" onClick={onCancel}>Cancel</button>
        <button className="btn-danger-sm" onClick={onConfirm} autoFocus>Delete</button>
      </div>
    </div>
  );
}

// ---------- typing / rendering ----------

function TypingThenContent({ content }) {
  const [shown, setShown] = React.useState("");
  const [done, setDone] = React.useState(false);

  React.useEffect(() => {
    setShown("");
    setDone(false);
    let i = 0;
    const step = () => {
      if (i >= content.length) { setDone(true); return; }
      i = Math.min(content.length, i + 2 + Math.floor(Math.random() * 5));
      setShown(content.slice(0, i));
      setTimeout(step, 24);
    };
    const t = setTimeout(step, 220);
    return () => clearTimeout(t);
  }, [content]);

  if (!shown) {
    return <div className="typing"><span></span><span></span><span></span></div>;
  }
  return (
    <>
      <Rendered content={shown} />
      {!done && <span style={{ display: "inline-block", width: 8, height: 16, background: "var(--ink-900)", marginLeft: 2, verticalAlign: "-3px", animation: "blink 0.9s steps(2) infinite" }}></span>}
    </>
  );
}

function Rendered({ content }) {
  const parts = [];
  const lines = content.split("\n");
  let block = [];
  let inCode = false;
  let codeBuf = [];

  const flushBlock = () => {
    if (!block.length) return;
    parts.push({ type: "p", text: block.join("\n") });
    block = [];
  };
  const flushCode = () => {
    parts.push({ type: "code", text: codeBuf.join("\n") });
    codeBuf = [];
  };

  for (const line of lines) {
    if (line.trim().startsWith("```")) {
      if (inCode) { flushCode(); inCode = false; }
      else { flushBlock(); inCode = true; }
      continue;
    }
    if (inCode) { codeBuf.push(line); continue; }
    if (line.trim() === "") { flushBlock(); continue; }
    block.push(line);
  }
  flushBlock();
  if (inCode) flushCode();

  return (
    <>
      {parts.map((p, i) => {
        if (p.type === "code") return <pre key={i}><code>{p.text}</code></pre>;
        return <p key={i} dangerouslySetInnerHTML={{ __html: inlineFormat(p.text) }} />;
      })}
    </>
  );
}

function inlineFormat(text) {
  return text
    .replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;")
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
    .replace(/`([^`]+)`/g, "<code>$1</code>");
}

window.Message = Message;
