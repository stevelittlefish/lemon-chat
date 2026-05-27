// App.jsx — root of the chat UI kit.

const SEED_CHATS = [
  {
    id: "c1",
    title: "Lemon AI roadmap",
    today: true,
    messages: [
      { id: "m1", role: "user",      content: "What should we ship in the next release?" },
      { id: "m2", role: "assistant", content: "Three ideas worth considering, in order of impact:\n\n**1. Conversation search.** People accumulate hundreds of chats. A keyword search across all of them — even a simple substring match — would be a big quality-of-life win.\n\n**2. Model presets.** A small set of named configurations (`creative`, `precise`, `coding`) that bundle temperature, top-p, and a system prompt. Saves users from learning what those knobs even do.\n\n**3. Drag-and-drop file attachments.** PDFs, text files, code. Just drop into the composer. Most of your competitors do this; the workflow is well-understood.\n\nWant me to write a quick spec for any of these?" },
    ],
  },
  {
    id: "c2",
    title: "Trip planning · Lisbon",
    today: true,
    messages: [
      { id: "m3", role: "user",      content: "Three days in Lisbon, low-key — what should we do?" },
      { id: "m4", role: "assistant", content: "Lisbon rewards walking and eating slowly.\n\n- **Day 1:** Stay in Alfama. Wander up to the castle in the late afternoon when the light is best. Eat at A Tasca do Chico — small, loud, fado.\n- **Day 2:** Tram 28 to Estrela. The park is quiet on weekday mornings. Lunch at Cervejaria Ramiro if you eat shellfish, otherwise Time Out Market.\n- **Day 3:** Day trip to Sintra. Take the train from Rossio. Pena Palace is touristy but worth it once.\n\nWhat's your budget like?" },
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

  const handleSend = (text) => {
    if (!active) return;
    const userMsg  = { id: "u" + Date.now(), role: "user", content: text };
    const replyId  = "a" + Date.now();
    const reply    = CANNED_REPLIES[Math.floor(Math.random() * CANNED_REPLIES.length)];
    const aiMsg    = { id: replyId, role: "assistant", content: reply, streaming: true };

    updateChat(active.id, (c) => {
      // Give the chat a real title from the first user message
      const title = (c.messages.length === 0 && text.length > 0)
        ? text.slice(0, 40).replace(/\s+\S*$/, '') + (text.length > 40 ? '…' : '')
        : c.title;
      return { ...c, title, messages: [...c.messages, userMsg, aiMsg] };
    });

    // After "streaming" completes (~ length-based), flip the streaming flag off
    const dur = Math.min(6000, 600 + reply.length * 12);
    setTimeout(() => {
      updateChat(active.id, (c) => ({
        ...c,
        messages: c.messages.map(m => m.id === replyId ? { ...m, streaming: false } : m),
      }));
    }, dur);
  };

  const handleNew = () => {
    const id = "n" + Date.now();
    setChats(prev => [{ id, title: "New chat", today: true, messages: [] }, ...prev]);
    setActiveId(id);
  };

  const handleDelete = (id) => {
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

  return (
    <div className="app">
      <Sidebar
        chats={chats}
        activeId={activeId}
        onSelect={setActiveId}
        onNew={handleNew}
        onDelete={handleDelete}
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
