// App.jsx — root of the Complete (text-completion) surface.
// Layout ("rail" | "bar") comes from ?layout= so the same app can be shown
// in two variants side-by-side on the design canvas.

const estTokens = (s = "") => Math.max(0, Math.round(s.length / 4));

// Canned continuations for a live Run. Each begins as a fresh paragraph so
// the prompt/generation split lands on a clean markdown boundary.
const CONTINUATIONS = [
  `There are three reasons this matters more than it first appears.

**First, the economics flip.** When the marginal cost of a capability falls to near zero, the constraint moves from *can we build it* to *should we, and where*. That is a different kind of question, and most teams are not organized to answer it.

**Second, the defaults harden.** Whatever ships in the first credible version tends to become the reference everyone else reacts to. Early choices cast long shadows.

**Third, attention is the real budget.** Compute is rentable; judgment is not. The teams that win spend their scarce judgment on the few decisions that compound.

So the practical move is to pick the one irreversible decision in front of you and spend real time there — and let the reversible ones be fast.`,

  `To see why, it helps to write the relationship down explicitly. Let $r$ be the per-step growth rate and $t$ the number of steps. Then the value after $t$ steps is

$$V_t = V_0\\,(1 + r)^t$$

The counterintuitive part lives in the regime where $r$ is small but $t$ is large. A rate of $r = 0.01$ looks like rounding error, yet over $t = 365$ steps it compounds to

$$V_{365} = V_0\\,(1.01)^{365} \\approx 37.8\\,V_0$$

That is the whole argument for consistency over intensity: the exponent does the work, not the base.`,

  `Here is the smallest version that actually runs:

\`\`\`python
def complete(prompt, model, *, temperature=0.7, max_tokens=256):
    stream = model.generate(
        prompt,
        temperature=temperature,
        max_tokens=max_tokens,
    )
    for token in stream:
        yield token        # hand each token back as it arrives
\`\`\`

A few notes on the shape of this:

- Streaming is a *generator*, not a return value — the caller decides when to stop reading.
- \`temperature\` and \`max_tokens\` are keyword-only on purpose; positional sampling args are how bugs get in.
- Everything else (stop sequences, top-p) layers on top without changing this contract.`,

  `The room had the particular stillness of places that are usually loud. Chairs pushed in. A whiteboard still ghosted with yesterday's diagram, half-erased, the arrows pointing at nothing now.

She set her coffee down and didn't drink it. Outside, the city was doing its early-morning impression of calm — delivery trucks, a street sweeper, the long exhale before everyone arrived and the day became real.

It would be a good day or it wouldn't. There was, she had learned, remarkably little in between, and almost none of it was decided in advance.`,
];

const SEED_SAVED = [
  {
    id: "s1",
    title: "Bull case for small models",
    meta: "llama-3.1-8b · 168 tok",
    prompt: "Make the case that small, local language models matter more than the frontier race suggests.\n",
    completion: CONTINUATIONS[0],
  },
  {
    id: "s2",
    title: "Why compounding feels like magic",
    meta: "mistral-7b · 121 tok",
    prompt: "Explain, with the actual math, why small consistent gains beat occasional big ones.\n",
    completion: CONTINUATIONS[1],
  },
  {
    id: "s3",
    title: "Minimal streaming completion",
    meta: "qwen2.5-coder-7b · 96 tok",
    prompt: "Write the smallest Python function that streams tokens from a model.\n",
    completion: CONTINUATIONS[2],
  },
  {
    id: "s4",
    title: "Cold open — empty office",
    meta: "phi-3.5-mini · 110 tok",
    prompt: "A short, quiet cold open. Someone arrives first at an office before a big day.\n",
    completion: CONTINUATIONS[3],
  },
];

function RunBtn({ streaming, canRun, onRun, onStop }) {
  if (streaming) {
    return (
      <button className="run-btn stop" onClick={onStop}>
        <Icon name="stop" size={13} /> Stop
      </button>
    );
  }
  return (
    <button className="run-btn" onClick={onRun} disabled={!canRun}>
      <Icon name="play" size={13} /> Run
    </button>
  );
}

function App() {
  const layout = (new URLSearchParams(location.search).get("layout") === "bar") ? "bar" : "rail";

  const [text, setText]         = React.useState(SEED_SAVED[0].prompt + SEED_SAVED[0].completion);
  const [mode, setMode]         = React.useState("read");
  const [streaming, setStreaming] = React.useState(false);
  const [genStart, setGenStart] = React.useState(SEED_SAVED[0].prompt.length);
  const [settled, setSettled]   = React.useState(true);
  const [activeId, setActiveId] = React.useState("s1");

  const [model, setModel]       = React.useState(MODELS[0]);
  const [temp, setTemp]         = React.useState(0.7);
  const [maxTokens, setMaxTokens] = React.useState(512);
  const [stops, setStops]       = React.useState(["\n\n"]);

  const taRef = React.useRef(null);
  const timer = React.useRef(null);
  const settleTimer = React.useRef(null);
  const contIdx = React.useRef(0);

  const clearTimers = () => {
    if (timer.current) { clearInterval(timer.current); timer.current = null; }
    if (settleTimer.current) { clearTimeout(settleTimer.current); settleTimer.current = null; }
  };
  React.useEffect(() => () => clearTimers(), []);

  const canRun = text.trim().length > 0 && !streaming;

  const finish = () => {
    clearTimers();
    setStreaming(false);
    settleTimer.current = setTimeout(() => setSettled(true), 900);
  };

  const run = () => {
    if (text.trim().length === 0 || streaming) return;
    clearTimers();
    setActiveId(null);

    const base = (text.length === 0 || text.endsWith("\n")) ? text : text + "\n\n";
    const cont = CONTINUATIONS[contIdx.current % CONTINUATIONS.length];
    contIdx.current += 1;
    const full = base + cont;
    const start = base.length;

    // Where generation stops: end of text, a stop sequence, or the token budget.
    let cutoff = full.length;
    for (const s of stops) {
      if (!s) continue;
      const idx = full.indexOf(s, start);
      if (idx !== -1) cutoff = Math.min(cutoff, idx);
    }
    cutoff = Math.min(cutoff, start + maxTokens * 4);

    setMode("read");
    setSettled(false);
    setGenStart(start);
    setStreaming(true);
    setText(base);

    let i = start;
    timer.current = setInterval(() => {
      i = Math.min(cutoff, i + (3 + Math.floor(Math.random() * 4)));
      setText(full.slice(0, i));
      if (i >= cutoff) finish();
    }, 16);
  };

  const stop = () => finish();

  const onChangeText = (v) => {
    setText(v);
    setGenStart(null);
    setActiveId(null);
  };

  const switchMode = (m) => {
    if (streaming) return;
    if (m === "write") {
      setGenStart(null);   // accept the generation into the editable document
      setTimeout(() => taRef.current?.focus(), 0);
    }
    setMode(m);
  };

  const newCompletion = () => {
    clearTimers();
    setStreaming(false);
    setText("");
    setGenStart(null);
    setSettled(true);
    setActiveId(null);
    setMode("write");
    setTimeout(() => taRef.current?.focus(), 0);
  };

  const loadSaved = (id) => {
    const s = SEED_SAVED.find(x => x.id === id);
    if (!s) return;
    clearTimers();
    setStreaming(false);
    setText(s.prompt + s.completion);
    setGenStart(s.prompt.length);
    setSettled(true);
    setMode("read");
    setActiveId(id);
  };

  // ⌘/Ctrl + Enter runs from anywhere.
  React.useEffect(() => {
    const onKey = (e) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
        e.preventDefault();
        if (!streaming && text.trim().length > 0) run();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  const addStop = (s) => setStops(prev => prev.includes(s) ? prev : [...prev, s]);
  const removeStop = (i) => setStops(prev => prev.filter((_, idx) => idx !== i));

  const chars = text.length;
  const toks  = estTokens(text);

  const paramProps = {
    model, onModel: setModel,
    temp, onTemp: setTemp,
    maxTokens, onMaxTokens: setMaxTokens,
    stops, onAddStop: addStop, onRemoveStop: removeStop,
  };

  const header = (
    <header className="cmp-head">
      <div className="titlewrap">
        <span className="eyebrow">Completion</span>
        <span className="ti">{activeId ? (SEED_SAVED.find(s => s.id === activeId)?.title) : "Untitled completion"}</span>
      </div>
      <div className="spacer"></div>
      <div className="cmp-counts">
        <span><b>{chars.toLocaleString()}</b> chars</span>
        <span><b>{toks.toLocaleString()}</b> tokens</span>
      </div>
      <div className="rw-toggle">
        <button className={mode === "write" ? "on" : ""} disabled={streaming} onClick={() => switchMode("write")}>
          <Icon name="pencil" size={14} /> Write
        </button>
        <button className={mode === "read" ? "on" : ""} onClick={() => switchMode("read")}>
          <Icon name="eye" size={14} /> Read
        </button>
      </div>
    </header>
  );

  const runHint = (
    <span className="run-hint"><kbd>⌘</kbd><kbd>↵</kbd> to run</span>
  );

  return (
    <div className={"cmp-app layout-" + layout}>
      <Sidebar
        saved={SEED_SAVED}
        activeId={activeId}
        onSelect={loadSaved}
        onNew={newCompletion}
        chatHref="../chat/index.html"
      />

      <main className="cmp-main">
        {header}
        <Editor
          text={text}
          mode={mode}
          streaming={streaming}
          genStart={genStart}
          settled={settled}
          onChange={onChangeText}
          taRef={taRef}
        />
        {layout === "rail" ? (
          <div className="cmp-actionbar">
            <div className="grow"></div>
            {runHint}
            <RunBtn streaming={streaming} canRun={canRun} onRun={run} onStop={stop} />
          </div>
        ) : (
          <ParamsBar {...paramProps}>
            {runHint}
            <RunBtn streaming={streaming} canRun={canRun} onRun={run} onStop={stop} />
          </ParamsBar>
        )}
      </main>

      {layout === "rail" && <ParamsRail {...paramProps} />}
    </div>
  );
}

window.App = App;
