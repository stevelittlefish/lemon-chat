// Thread.jsx — message list + empty-state with suggested prompts.

function Thread({ messages, onSuggest, modelName, lastAssistantId, onDeleteMsg, onEditMsg, onRegenerate }) {
  const scrollRef = React.useRef();

  React.useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages.length, messages[messages.length - 1]?.content?.length]);

  if (messages.length === 0) {
    return (
      <div className="thread-scroll" ref={scrollRef}>
        <div className="empty-thread">
          <img src="../../assets/lemon-slice.svg" alt="" />
          <h2>What can we help with?</h2>
          <p>You're chatting with <strong>{modelName}</strong> on this machine. Nobody else can see this.</p>
          <div className="empty-suggestions">
            <button onClick={() => onSuggest("Summarize this article for me…")}>
              Summarize an article
            </button>
            <button onClick={() => onSuggest("Help me rewrite this email.")}>
              Rewrite an email
            </button>
            <button onClick={() => onSuggest("Write a Python script to rename files.")}>
              Write a small script
            </button>
            <button onClick={() => onSuggest("What's a good weekend project?")}>
              Brainstorm an idea
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="thread-scroll" ref={scrollRef}>
      <div className="thread">
        {messages.map(m => (
          <Message
            key={m.id}
            msg={m}
            isLastAssistant={m.id === lastAssistantId}
            onDelete={() => onDeleteMsg(m.id)}
            onEdit={(t) => onEditMsg(m.id, t)}
            onRegenerate={() => onRegenerate(m.id)}
          />
        ))}
      </div>
    </div>
  );
}

window.Thread = Thread;
