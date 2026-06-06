# Frontend Code Review

## Bugs

### Context modal doesn't show tool-call message content
**File:** `static/js/thread.js:764-781`

When the "LLM context" modal renders messages, it does:
```js
pre.textContent = msg.content;
```

But for assistant messages that made tool calls, the server sets `content: ""` and puts all the data in `tool_calls`. The context viewer displays these as empty `<pre>` blocks. The model saw a structured tool-call message; the user sees a blank entry. The modal should also render `msg.tool_calls` when present.

---

### `handlePickerSelect` catch block comment is wrong and state can be inconsistent
**File:** `static/js/app.js:75-89`

```js
try {
    const conv = await convApi.create(null, null, sel.id);
    // ... set state ...
    await applyFirstMessage(conv.id, null);
} catch {
    // Creation failed — stay on picker
}
```

`applyFirstMessage` is inside the try block. If it throws after the conversation has been created, the catch label "Creation failed — stay on picker" is wrong — the conversation exists in the DB and sidebar but the UI reverts to the picker. Since `applyFirstMessage` internally catches its own errors, this won't fire in practice, but the structure is misleading and one future refactor away from being a real bug.

---

### `sidebar.addItem` doesn't prevent duplicates
**File:** `static/js/sidebar.js:275-286`

`state.items.unshift(item)` with no deduplication check. If called twice for the same item (race between WS `conversations_changed` event firing `sidebar.load()` and a local `addItem` call), you get duplicate entries in `state.items` and two sidebar rows for the same conversation. The subsequent `renderList()` from `load()` would fix the display, but state would be corrupt until then.

---

### `header.js` loses the open dropdown on any title update
**File:** `static/js/header.js:80-110`

`render()` does `headerEl.innerHTML = ...` on every call, which destroys all child elements including any open dropdown. A WebSocket title update during model selection causes the dropdown to vanish with no user feedback. `updateTitle()` (line 32-36) already has an incremental path for title changes — `render()` should never be called when only the title changes.

---

### `complete-app.js` has a potential race in `finishStreaming`
**File:** `static/js/complete-app.js:483-527`

After streaming ends, the function fetches the completion from the server:
```js
const comp = await completionsApi.get(activeCompletionId);
if (comp.id === activeCompletionId && comp.content != null) { ... }
```

The check `comp.id === activeCompletionId` guards against having switched to a different completion. But it checks `comp.id` against `activeCompletionId` — if the user navigates to completion ID 5, then the in-flight GET for the previous completion ID 5 (if IDs were reused, which won't happen with autoincrement, but still) would pass the guard. More realistically: `activeCompletionId` is updated synchronously during navigation but the stale GET resolves later and overwrites `currentText` with old data. The guard should capture the ID in a closure before the await.

---

## Duplicated Code

### SSE streaming code is copy-pasted between `messages.send` and `completions.run`
**File:** `static/js/api.js:79-122, 152-188`

Both functions contain identical boilerplate:
```js
const reader = res.body.getReader();
const decoder = new TextDecoder();
let buffer = '';
while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop();
    for (const line of lines) {
        if (!line.startsWith('data: ')) continue;
        const payload = line.slice(6);
        if (payload === '[DONE]') { onDone?.(); return; }
        try { ... } catch {}
    }
}
onDone?.();
```

The only difference is which fields are destructured from the parsed JSON. A shared `consumeSSE(res, onEvent)` helper that handles the read loop and calls `onEvent(parsed)` for each chunk would eliminate ~50 lines of duplicate fragile code.

---

### Modal construction pattern repeated four times
**Files:** `static/js/thread.js`, `static/js/sidebar.js`

`getMdModal()`, `getForkModal()`, `getCtxModal()`, and `getDeleteModal()` all implement identical lazy-singleton construction:
```js
let _xModal = null;
function getXModal() {
    if (_xModal) return _xModal;
    // ... ~20-40 lines of createElement + wiring ...
    overlay.addEventListener('mousedown', e => { if (e.target === overlay) closeXModal(); });
    document.addEventListener('keydown', e => { if (e.key === 'Escape' && overlay.classList.contains('open')) closeXModal(); });
    document.body.appendChild(overlay);
    _xModal = { overlay, ... };
    return _xModal;
}
```

This suggests an `createModal({ title, body, actions })` factory or a small `Modal` class that handles the overlay, Escape key, and backdrop click once.

Note: the keyboard Escape listener is added to `document` every time a new modal type is initialized. They accumulate across the lifetime of the page (4 listeners total, each checking their own class). This works but is untidy.

---

### `handleUndo` and `handleRedo` are 95% identical
**File:** `static/js/complete-app.js:569-608`

Both functions:
1. Guard on `prevContent != null` and `undone` flag
2. Hide `undoBtnEl`
3. Call the API (the only real difference)
4. Update `currentText`, `prevContent`, `undone`
5. Set `genStart = null; settled = true`
6. Set `textareaEl.value = currentText`
7. Call `setMode('write')`
8. Call `renderControls()`
9. Call `updateUndoBtn()`
10. `setTimeout` to scroll

They could share a `performUndoRedo(forward: boolean)` function.

---

### `applyFirstMessage` called from two different flows with similar but not identical semantics
**File:** `static/js/app.js:273-288`

`handlePickerSelect` calls it with `charId = null` (use conversation's character). `sendMessage` calls it with `charId = null` when `charId !== null` was used to create the conversation. `handleSelectionChange` calls it with `sel.id`. The function is flexible enough but the callers each have slightly different surrounding state management (when `sidebar.updateItem` is called, when `applyAvatarContext` is called) — fragile to extend.

---

## Confusing / Poorly Structured Code

### `startInlineEdit` in sidebar manually transplants DOM nodes
**File:** `static/js/sidebar.js:70-138`

To convert the `<a>` sidebar item to an editable `<input>`, the code:
1. Creates a `<div>`, moves all children from `<a>` into it, replaces `<a>` with `<div>`
2. Swaps the title span for an `<input>`
3. On commit: swaps `<input>` back, creates a new `<a>`, moves all children back, re-attaches event listeners

This is a fragile DOM surgery. If any child element (e.g. the menu button) has its own event listeners, they survive the move fine in modern browsers, but the comment `// Swap <a> → <div> so the href cannot interfere with click-to-reposition` buries a subtle problem. A simpler approach: just `contenteditable` the title span in place, or use a positioned `<input>` overlay.

---

### `complete-app.js` manages mode state across 7 module-level variables
**File:** `static/js/complete-app.js:17-32`

```js
let currentText = '';
let mode = 'write';
let streaming = false;
let genStart = null;
let settled = true;
let prevContent = null;
let undone = false;
```

These interact in complex ways: `mode` controls which DOM elements are shown, `genStart` determines where the prompt/generated split is, `settled` controls the highlight animation, `undone` controls the undo/redo button label. `loadCompletion` resets all 7 to initial values (lines 398-405). Any new state variable added later must also be reset there — easy to forget.

A `CompletionEditorState` object with a `reset()` method would make the coupling explicit and easier to maintain.

---

### `thread.js` module has too many responsibilities
**File:** `static/js/thread.js`

The file handles:
- Message rendering (buildMessage, renderMessages, appendMessage)
- Streaming controller (startStreaming)
- Artifact panel (openArtifactPanel, closeArtifactPanel)
- Four separate modal implementations (markdown, fork, context, info popup)
- Avatar management (buildAvatar, setAvatarContext)
- Scroll management
- Tool call display (buildToolCallEl, buildToolCallSection)

At ~930 lines, finding the code for any one concern requires knowing the file's internal layout. The artifact panel, modals, and tool call components could each be separate modules.

---

### `withRetry` is used as a side-effect function but returns the element
**File:** `static/js/thread.js:386-393`

```js
function withRetry(img) {
    img.onerror = () => { ... };
    return img;
}
```

Called in two styles: `const img = withRetry(document.createElement('img'))` (uses return value) and `threadEl.querySelectorAll('.picker-card-avatar img').forEach(withRetry)` (discards return value). The dual-use is workable but the function name doesn't communicate that it mutates in place — `addRetryBehavior(img)` returning void would be clearer.

---

### `header.js` rebuilds innerHTML for every state change but also has incremental updates
**File:** `static/js/header.js:80, 32-36, 72-77`

`render()` writes the entire `headerEl.innerHTML`. But `updateTitle()` and `setSelection()` do targeted DOM mutations. This means two code paths for the same concern, and a caller who calls `render()` instead of `updateTitle()` would drop any programmatically-applied classes on the title element. The pattern should be one or the other — either always full re-render, or always targeted updates.

---

## Maintenance Concerns

### The `onNewTurn` handler in api.js is referenced but not declared in the destructured params
**File:** `static/js/api.js:73, 113`

```js
send: (conversationId, content, selection, { onName, onDelta, onDone, onAborted, onError, onStats, onMessageId, onToolCall, onToolResult, onAttachment }) => {
```

`onNewTurn` is not in the destructured parameter list, but it's used on line 113:
```js
if (new_turn) onNewTurn?.(new_turn);
```

Because `onNewTurn` is used with optional chaining (`?.`), this silently does nothing rather than throwing. The new turn event is dropped if the caller passes `onNewTurn` — except it actually works because `onNewTurn` in app.js is passed and JavaScript closures capture the outer scope variable. Wait, no — `onNewTurn` is not in the destructured params, so it's `undefined` inside the function, and `onNewTurn?.()` is always a no-op regardless of what the caller passes. The new-turn callback never fires.

Actually looking again at app.js line 233: the caller does pass `onNewTurn` in the options object. But because it's not destructured in `send`'s parameter list, it's silently ignored. This is a real bug: `stream = thread.startStreaming()` is never called at the start of a second tool-call turn, so all subsequent responses stream into the same bubble.

Wait — re-reading more carefully. The destructured params in api.js line 73 don't include `onNewTurn`. But in JavaScript, you can still access the rest of the options object if you capture it. The function doesn't, so `onNewTurn` is genuinely inaccessible. This is a real bug that prevents multi-turn tool call UI from working correctly.

---

### Escape key listeners accumulate on `document` as modals are first opened
**File:** `static/js/thread.js`, `static/js/sidebar.js`

Each modal adds one `document.addEventListener('keydown', ...)` listener the first time it's opened (inside the `getXModal()` lazy initializer). These listeners are never removed. After opening all modals once, there are 4 permanent Escape-key listeners on the document. Each checks its own modal's open state, so they don't interfere, but it's a resource leak pattern.

---

### No loading/error state shown when conversation fails to load
**File:** `static/js/app.js:184-191`

```js
} catch {
    activeConversationId = null;
    thread.setConversationId(null);
    sidebar.setActive(null);
    history.replaceState({ conversationId: null }, '', '/');
    showPickerScreen();
}
```

If a conversation fails to load (network error, 404), the URL changes to `/` and the picker is shown with no error message. The user has no feedback about why the conversation they clicked on is gone.

---

### Settings pages import api.js directly and duplicate auth-check logic

Each settings page (`settings-account.js`, `settings-character-edit.js`, etc.) independently calls `auth.me()` and redirects to `/` on failure. This is a manual guard on every page. A shared `requireAuth()` module function that throws/redirects and can be awaited at page start would centralize this.

---

### `complete-app.js` uses `console.error` throughout instead of user-visible errors
**File:** `static/js/complete-app.js:431, 444, 479, etc.**

Most async failures in the completions page log to `console.error` but show nothing to the user. The `handleSave` failure, model change failure, and run errors are invisible in the UI. The chat page has a `stream.error(...)` path that shows errors inline — completions has no equivalent fallback display.
