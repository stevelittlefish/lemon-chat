// App.jsx — root of the chat UI kit.

// Loose-but-realistic token estimator (chars/4, clamped).
const estTokens = (s = "") => Math.max(1, Math.round(s.length / 4));

// Pretend-metrics for a freshly-generated assistant message.
function fakeMetrics(promptText, replyText) {
  const promptTokens   = estTokens(promptText);
  const responseTokens = estTokens(replyText);
  // Loose realistic speeds: 30–95 tok/s for a local model on a laptop.
  const tps      = 30 + Math.random() * 65;
  const genMs    = (responseTokens / tps) * 1000;
  const firstTok = 180 + Math.random() * 420;  // time-to-first-token
  const totalMs  = Math.round(firstTok + genMs);
  return {
    promptTokens,
    responseTokens,
    totalMs,
    tokensPerSec: +(responseTokens / (genMs / 1000)).toFixed(1),
    ttftMs:       Math.round(firstTok),
  };
}

const SEED_CHATS = [
  {
    id: "c1",
    title: "Lemon AI roadmap",
    today: true,
    messages: [
      { id: "m1", role: "user",      content: "What should we ship in the next release?" },
      { id: "m2", role: "assistant", content: "Three ideas worth considering, in order of impact:\n\n**1. Conversation search.** People accumulate hundreds of chats. A keyword search across all of them — even a simple substring match — would be a big quality-of-life win.\n\n**2. Model presets.** A small set of named configurations (`creative`, `precise`, `coding`) that bundle temperature, top-p, and a system prompt. Saves users from learning what those knobs even do.\n\n**3. Drag-and-drop file attachments.** PDFs, text files, code. Just drop into the composer. Most of your competitors do this; the workflow is well-understood.\n\nWant me to write a quick spec for any of these?",
        metrics: { promptTokens: 11, responseTokens: 142, totalMs: 2380, tokensPerSec: 64.2, ttftMs: 240 } },
    ],
  },
  {
    id: "c2",
    title: "Trip planning · Lisbon",
    today: true,
    messages: [
      { id: "m3", role: "user",      content: "Three days in Lisbon, low-key — what should we do?" },
      { id: "m4", role: "assistant", content: "Lisbon rewards walking and eating slowly.\n\n- **Day 1:** Stay in Alfama. Wander up to the castle in the late afternoon when the light is best. Eat at A Tasca do Chico — small, loud, fado.\n- **Day 2:** Tram 28 to Estrela. The park is quiet on weekday mornings. Lunch at Cervejaria Ramiro if you eat shellfish, otherwise Time Out Market.\n- **Day 3:** Day trip to Sintra. Take the train from Rossio. Pena Palace is touristy but worth it once.\n\nWhat's your budget like?",
        metrics: { promptTokens: 14, responseTokens: 118, totalMs: 1910, tokensPerSec: 71.5, ttftMs: 220 } },
    ],
  },
  {
    id: "c3", title: "Recipe ideas",                today: false, messages: [],
  },
  {
    id: "c4", title: "Python script to rename files", today: false, messages: [],
  },
  {
    id: "c5", title: "Notes from book club",        today: false, messages: [],
  },
];

const CANNED_REPLIES = [
  "Happy to help with that. Walk me through what you have so far?",
  "Here's how I'd approach it:\n\n- Start small. Pick the one piece that you're most uncertain about.\n- Sketch it out before writing real code.\n- Get feedback early — even if it feels half-baked.\n\nDoes that fit the shape of what you're working on?",
  "Sure. The shortest version:\n\n```python\nimport os\nfor f in os.listdir('.'):\n    if f.endswith('.txt'):\n        os.rename(f, f.replace(' ', '_'))\n```\n\nReplaces spaces with underscores in every `.txt` file in the current directory. Let me know what you actually want to rename and I'll tighten this up.",
  "Good question. The short answer is: it depends on what you mean by **best**. If you mean fastest, that's one thing. If you mean most accurate, that's another. Which one matters more here?",
];

function App() {
  const [chats, setChats] = React.useState(SEED_CHATS);
  const [activeId, setActiveId] = React.useState(SEED_CHATS[0].id);
  const [model, setModel] = React.useState(MODELS[0]);
  const [pendingInput, setPendingInput] = React.useState("");

  const active = chats.find(c => c.id === activeId);

  const updateChat = (id, fn) => {
    setChats(prev => prev.map(c => c.id === id ? fn(c) : c));
  };

  // Append a user message + a streaming assistant reply.
  const appendExchange = (chatId, userText, { skipUserMsg = false } = {}) => {
    const userMsg  = { id: "u" + Date.now(), role: "user", content: userText };
    const replyId  = "a" + Date.now() + Math.floor(Math.random() * 1000);
    const reply    = CANNED_REPLIES[Math.floor(Math.random() * CANNED_REPLIES.length)];
    const metrics  = fakeMetrics(userText, reply);
    const aiMsg    = { id: replyId, role: "assistant", content: reply, streaming: true, metrics };

    updateChat(chatId, (c) => {
      const title = (c.messages.length === 0 && userText.length > 0 && !skipUserMsg)
        ? userText.slice(0, 40).replace(/\s+\S*$/, '') + (userText.length > 40 ? '…' : '')
        : c.title;
      const next = skipUserMsg ? [...c.messages, aiMsg] : [...c.messages, userMsg, aiMsg];
      return { ...c, title, messages: next };
    });

    const dur = Math.min(6000, 600 + reply.length * 12);
    setTimeout(() => {
      updateChat(chatId, (c) => ({
        ...c,
        messages: c.messages.map(m => m.id === replyId ? { ...m, streaming: false } : m),
      }));
    }, dur);
  };

  const handleSend = (text) => {
    if (!active) return;
    appendExchange(active.id, text);
  };

  const handleNew = () => {
    const id = "n" + Date.now();
    setChats(prev => [{ id, title: "New chat", today: true, messages: [] }, ...prev]);
    setActiveId(id);
  };

  const handleDeleteChat = (id) => {
    setChats(prev => prev.filter(c => c.id !== id));
    if (id === activeId) {
      const remaining = chats.filter(c => c.id !== id);
      if (remaining.length) setActiveId(remaining[0].id);
      else handleNew();
    }
  };

  const handleRename = (id) => {
    const c = chats.find(c => c.id === id);
    const next = prompt("Rename chat", c.title);
    if (next != null) updateChat(id, (c) => ({ ...c, title: next.trim() || c.title }));
  };

  // ---------- Per-message actions ----------

  const handleDeleteMsg = (msgId) => {
    updateChat(activeId, (c) => ({
      ...c,
      messages: c.messages.filter(m => m.id !== msgId),
    }));
  };

  const handleEditMsg = (msgId, newContent) => {
    // Truncate to (and including) the edited message, then regenerate.
    const chat = chats.find(c => c.id === activeId);
    if (!chat) return;
    const idx = chat.messages.findIndex(m => m.id === msgId);
    if (idx === -1) return;

    const edited = { ...chat.messages[idx], content: newContent };
    const kept   = [...chat.messages.slice(0, idx), edited];

    updateChat(activeId, (c) => ({ ...c, messages: kept }));

    // If the edited message was a user message, fire a fresh reply.
    if (edited.role === "user") {
      // Defer so the state has settled before we push the assistant msg.
      setTimeout(() => appendExchange(activeId, newContent, { skipUserMsg: true }), 0);
    }
  };

  const handleRegenerate = (msgId) => {
    const chat = chats.find(c => c.id === activeId);
    if (!chat) return;
    const idx = chat.messages.findIndex(m => m.id === msgId);
    if (idx < 1) return;
    const prevUser = [...chat.messages.slice(0, idx)].reverse().find(m => m.role === "user");
    if (!prevUser) return;

    // Drop the old assistant message and anything after, then re-run.
    updateChat(activeId, (c) => ({ ...c, messages: c.messages.slice(0, idx) }));
    setTimeout(() => appendExchange(activeId, prevUser.content, { skipUserMsg: true }), 0);
  };

  // Find the id of the latest assistant message (used to mark which one
  // gets the always-visible footer + info button).
  const lastAssistantId = (() => {
    const ms = active?.messages || [];
    for (let i = ms.length - 1; i >= 0; i--) {
      if (ms[i].role === "assistant") return ms[i].id;
    }
    return null;
  })();

  return (
    <div className="app">
      <Sidebar
        chats={chats}
        activeId={activeId}
        onSelect={setActiveId}
        onNew={handleNew}
        onDelete={handleDeleteChat}
        onRename={handleRename}
      />
      <main className="main">
        <ChatHeader
          title={active?.title || "Untitled"}
          onTitleChange={(t) => updateChat(activeId, (c) => ({ ...c, title: t }))}
          model={model}
          onModelChange={setModel}
        />
        <Thread
          messages={active?.messages || []}
          onSuggest={(s) => setPendingInput({ text: s, key: Date.now() })}
          modelName={model.name}
          lastAssistantId={lastAssistantId}
          onDeleteMsg={handleDeleteMsg}
          onEditMsg={handleEditMsg}
          onRegenerate={handleRegenerate}
        />
        <Composer
          onSend={handleSend}
          initial={pendingInput?.text}
        />
      </main>
    </div>
  );
}

window.App = App;
