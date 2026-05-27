// Message.jsx — a single message bubble (user or AI).
// AI bubbles render canned markdown-ish content with bold/code/pre.

function Message({ msg }) {
  const isUser = msg.role === "user";
  return (
    <div className={"msg " + (isUser ? "user" : "ai")}>
      <div className="meta">{isUser ? "you" : "lemon"}</div>
      <div className={"bubble " + (isUser ? "user" : "ai")}>
        {msg.streaming ? <TypingThenContent content={msg.content} /> : <Rendered content={msg.content} />}
      </div>
      {!isUser && !msg.streaming && (
        <div className="actions">
          <button title="Copy"><Icon name="copy" size={14} /></button>
          <button title="Regenerate"><Icon name="refresh" size={14} /></button>
          <button title="Good response"><Icon name="thumbsUp" size={14} /></button>
        </div>
      )}
    </div>
  );
}

function TypingThenContent({ content }) {
  const [shown, setShown] = React.useState("");
  const [done, setDone] = React.useState(false);

  React.useEffect(() => {
    setShown("");
    setDone(false);
    let i = 0;
    const step = () => {
      if (i >= content.length) { setDone(true); return; }
      // chunk of 2-6 chars
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

// Very small markdown-ish renderer (paragraphs, **bold**, `code`, fenced ```)
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
