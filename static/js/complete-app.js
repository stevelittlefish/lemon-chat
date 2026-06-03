import { auth, models as modelApi, completions as completionsApi } from './api.js';
import * as sidebar from './sidebar.js';
import * as header from './header.js';
import * as ws from './ws.js';
import { preload as preloadIcons, icon } from './icons.js';
import { render as renderMarkdown } from './markdown.js';

const appEl = document.getElementById('app');
const completeMainEl = document.getElementById('complete-main');
const completeEmptyEl = document.getElementById('complete-empty');

let currentUser = null;
let activeCompletionId = null;
let modelList = [];

// Editor state
let currentText = '';
let mode = 'write';       // 'write' | 'read'
let streaming = false;
let genStart = null;      // char offset where generated text begins
let settled = true;       // lemon highlight faded
let prevContent = null;   // the "other" content snapshot (pre-run when undone=false, post-run when undone=true)
let undone = false;       // true when the last action was an undo (enabling redo)
let abortRun = null;      // fn to abort in-flight stream
let settleTimer = null;
let autoSaveTimer = null;

// Params state
let maxTokens = 512;
let temperature = 0.7;

// Token stats from last run
let lastPromptTokens = null;
let lastCompletionTokens = null;

// Editor DOM refs (created once)
let editorWrapEl = null;
let editorScrollEl = null;
let textareaEl = null;
let readPromptEl = null;
let readGenEl = null;
let caretEl = null;
let actionbarEl = null;
let undoBtnEl = null;
let runBtnEl = null;
let modeWriteBtnEl = null;
let modeReadBtnEl = null;

async function start() {
  try {
    currentUser = await auth.me();
  } catch {
    window.location.href = '/';
    return;
  }
  appEl.classList.remove('hidden');
  initApp();
}

async function initApp() {
  sidebar.init({
    username: currentUser.username,
    onSelect: loadCompletion,
    onNew: newCompletion,
    api: completionsApi,
    newLabel: 'New completion',
  });

  const [models] = await Promise.all([
    modelApi.list(),
    sidebar.load(),
    preloadIcons(),
  ]);
  modelList = models;

  header.init({
    models: modelList,
    characters: [],
    onChange: async (sel) => {
      if (!activeCompletionId || sel?.type !== 'model') return;
      try {
        await completionsApi.update(activeCompletionId, { model: sel.name });
      } catch (err) {
        console.error('failed to save model change:', err);
      }
    },
  });

  ws.on('completion_titled', ({ id, title }) => {
    sidebar.updateTitle(id, title);
    if (id === activeCompletionId) header.updateTitle(title);
  });
  ws.connect();

  createEditorDOM();

  window.addEventListener('popstate', (e) => {
    const id = e.state?.completionId ?? null;
    loadCompletion(id, { pushHistory: false });
  });

  const params = new URLSearchParams(location.search);
  const c = params.get('c');
  const initialId = c ? Number(c) : null;
  history.replaceState({ completionId: initialId }, '', location.href);
  if (initialId) {
    loadCompletion(initialId, { pushHistory: false });
  } else {
    showEmpty();
  }

  window.addEventListener('keydown', (e) => {
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      e.preventDefault();
      if (activeCompletionId && !streaming && currentText.trim().length > 0) handleRun();
    }
  });
}

function createEditorDOM() {
  // Outer wrapper (hidden until a completion is loaded)
  editorWrapEl = document.createElement('div');
  editorWrapEl.id = 'cmp-editor-wrap';
  editorWrapEl.className = 'cmp-editor-wrap hidden';

  // Content row: editor + params rail side by side
  const contentRowEl = document.createElement('div');
  contentRowEl.className = 'cmp-content-row';

  editorScrollEl = document.createElement('div');
  editorScrollEl.className = 'cmp-editor-scroll';

  const docEl = document.createElement('div');
  docEl.className = 'cmp-doc';

  textareaEl = document.createElement('textarea');
  textareaEl.id = 'cmp-textarea';
  textareaEl.className = 'cmp-write';
  textareaEl.placeholder = 'Start writing, then press Run — the model continues from where you stop…';
  textareaEl.spellcheck = false;
  textareaEl.addEventListener('input', onTextareaInput);

  readPromptEl = document.createElement('div');
  readPromptEl.className = 'cmp-read';

  readGenEl = document.createElement('div');
  readGenEl.className = 'cmp-read cmp-gen';

  caretEl = document.createElement('span');
  caretEl.className = 'cmp-caret';

  docEl.append(textareaEl, readPromptEl, readGenEl);
  editorScrollEl.appendChild(docEl);

  // Left column: editor scroll + action bar
  const editorColEl = document.createElement('div');
  editorColEl.className = 'cmp-editor-col';

  actionbarEl = document.createElement('div');
  actionbarEl.className = 'cmp-actionbar';

  undoBtnEl = document.createElement('button');
  undoBtnEl.className = 'cmp-undo-btn';
  undoBtnEl.classList.add('hidden');

  runBtnEl = document.createElement('button');
  runBtnEl.className = 'run-btn';

  const growEl = document.createElement('div');
  growEl.className = 'grow';

  const hintEl = document.createElement('span');
  hintEl.className = 'run-hint';
  hintEl.innerHTML = '<kbd>⌘</kbd><kbd>↵</kbd> to run';

  const rwToggleEl = document.createElement('div');
  rwToggleEl.className = 'rw-toggle';

  modeWriteBtnEl = document.createElement('button');
  modeReadBtnEl = document.createElement('button');

  rwToggleEl.append(modeWriteBtnEl, modeReadBtnEl);
  actionbarEl.append(undoBtnEl, growEl, hintEl, rwToggleEl, runBtnEl);
  editorColEl.append(editorScrollEl, actionbarEl);

  // Right rail: params
  const railEl = createRailDOM();

  contentRowEl.append(editorColEl, railEl);
  editorWrapEl.appendChild(contentRowEl);
  completeMainEl.appendChild(editorWrapEl);

  undoBtnEl.addEventListener('click', () => undone ? handleRedo() : handleUndo());
  runBtnEl.addEventListener('click', () => {
    if (streaming) handleStop();
    else handleRun();
  });
  modeWriteBtnEl.addEventListener('click', () => setMode('write'));
  modeReadBtnEl.addEventListener('click', () => setMode('read'));

  renderControls();
}

function createRailDOM() {
  const rail = document.createElement('aside');
  rail.className = 'cmp-rail';

  const head = document.createElement('div');
  head.className = 'rail-head';
  head.innerHTML = `<span class="rail-head-ico">${icon('sliders', 15)}</span> Parameters`;

  const body = document.createElement('div');
  body.className = 'rail-body';

  // Max tokens field
  const tokField = document.createElement('div');
  tokField.className = 'param-field';
  tokField.innerHTML = `<div class="param-label">Max tokens</div>`;
  const tokInput = document.createElement('input');
  tokInput.type = 'number';
  tokInput.className = 'param-num';
  tokInput.min = '1';
  tokInput.max = '32768';
  tokInput.step = '1';
  tokInput.value = maxTokens;
  tokInput.addEventListener('change', () => {
    const v = parseInt(tokInput.value, 10);
    if (!isNaN(v) && v > 0) maxTokens = v;
    tokInput.value = maxTokens;
  });
  tokField.appendChild(tokInput);

  // Temperature field
  const tempField = document.createElement('div');
  tempField.className = 'param-field';
  const tempValEl = document.createElement('span');
  tempValEl.className = 'param-val';
  tempValEl.textContent = temperature.toFixed(2);
  const tempLabelRow = document.createElement('div');
  tempLabelRow.className = 'param-label-row';
  tempLabelRow.innerHTML = `<span class="param-label">Temperature</span>`;
  tempLabelRow.appendChild(tempValEl);
  const tempRange = document.createElement('input');
  tempRange.type = 'range';
  tempRange.className = 'param-range';
  tempRange.min = '0';
  tempRange.max = '2';
  tempRange.step = '0.05';
  tempRange.value = temperature;
  tempRange.addEventListener('input', () => {
    temperature = parseFloat(tempRange.value);
    tempValEl.textContent = temperature.toFixed(2);
  });
  const tempHints = document.createElement('div');
  tempHints.className = 'param-range-hints';
  tempHints.innerHTML = '<span>precise</span><span>creative</span>';
  tempField.append(tempLabelRow, tempRange, tempHints);

  body.append(tokField, tempField);
  rail.append(head, body);
  return rail;
}

function onTextareaInput() {
  currentText = textareaEl.value;
  autoGrowTextarea();
  genStart = null;
  prevContent = null;
  undone = false;
  undoBtnEl.classList.add('hidden');
  scheduleAutoSave();
}

function scheduleAutoSave() {
  if (!activeCompletionId) return;
  if (autoSaveTimer) clearTimeout(autoSaveTimer);
  const savedId = activeCompletionId;
  const savedText = currentText;
  autoSaveTimer = setTimeout(async () => {
    autoSaveTimer = null;
    if (activeCompletionId !== savedId) return;
    try {
      await completionsApi.update(savedId, { content: savedText });
    } catch (err) {
      console.error('auto-save failed:', err);
    }
  }, 1000);
}

function autoGrowTextarea() {
  textareaEl.style.height = 'auto';
  textareaEl.style.height = textareaEl.scrollHeight + 'px';
}

function renderControls() {
  // Write/Read toggle buttons
  modeWriteBtnEl.innerHTML = icon('pencil', 14) + ' Write';
  modeWriteBtnEl.className = 'btn' + (mode === 'write' ? ' on' : '');
  modeWriteBtnEl.disabled = streaming;

  modeReadBtnEl.innerHTML = icon('eye', 14) + ' Read';
  modeReadBtnEl.className = 'btn' + (mode === 'read' ? ' on' : '');

  // Run / Stop button
  if (streaming) {
    runBtnEl.className = 'run-btn stop';
    runBtnEl.innerHTML = icon('square', 13) + ' Stop';
    runBtnEl.disabled = false;
  } else {
    runBtnEl.className = 'run-btn';
    runBtnEl.innerHTML = icon('play', 13) + ' Run';
    runBtnEl.disabled = !activeCompletionId || currentText.trim().length === 0;
  }
}

function setMode(newMode) {
  if (streaming && newMode === 'write') return;
  mode = newMode;
  if (mode === 'write') {
    genStart = null;
    textareaEl.classList.remove('hidden');
    readPromptEl.classList.add('hidden');
    readGenEl.classList.add('hidden');
    setTimeout(() => { textareaEl.focus(); autoGrowTextarea(); }, 0);
  } else {
    textareaEl.classList.add('hidden');
    readPromptEl.classList.remove('hidden');
    readGenEl.classList.toggle('hidden', genStart == null);
    updateReadView();
  }
  renderControls();
}

function updateReadView() {
  if (genStart != null) {
    readPromptEl.innerHTML = renderMarkdown(currentText.slice(0, genStart));
    readGenEl.innerHTML = renderMarkdown(currentText.slice(genStart));
    if (streaming) readGenEl.appendChild(caretEl);
    readGenEl.classList.toggle('settled', settled);
    readGenEl.classList.remove('hidden');
  } else {
    readPromptEl.innerHTML = renderMarkdown(currentText);
    readGenEl.classList.add('hidden');
    readGenEl.classList.remove('settled');
  }
}

function showEmpty() {
  if (autoSaveTimer) { clearTimeout(autoSaveTimer); autoSaveTimer = null; }
  activeCompletionId = null;
  streaming = false;
  completeEmptyEl.classList.remove('hidden');
  editorWrapEl?.classList.add('hidden');
  header.setConversation(null, null);
  sidebar.setActive(null);
}

function showEditor() {
  completeEmptyEl.classList.add('hidden');
  editorWrapEl.classList.remove('hidden');
}

async function loadCompletion(id, { pushHistory = true } = {}) {
  if (!id) {
    if (pushHistory) history.pushState({ completionId: null }, '', '/complete');
    showEmpty();
    return;
  }
  if (autoSaveTimer) { clearTimeout(autoSaveTimer); autoSaveTimer = null; }
  if (pushHistory) history.pushState({ completionId: id }, '', `/complete?c=${id}`);
  activeCompletionId = id;
  sidebar.setActive(id);
  const item = sidebar.getItem(id);
  header.setConversation(id, item?.title ?? null);
  showEditor();

  // Reset editor state for the new completion
  currentText = '';
  genStart = null;
  prevContent = null;
  undone = false;
  lastPromptTokens = null;
  lastCompletionTokens = null;
  settled = true;
  streaming = false;
  mode = 'write';
  undoBtnEl.classList.add('hidden');
  textareaEl.value = '';
  textareaEl.classList.remove('hidden');
  readPromptEl.classList.add('hidden');
  readGenEl.classList.add('hidden');
  renderControls();
  updateTokenStats();

  try {
    const comp = await completionsApi.get(id);
    if (comp.id !== activeCompletionId) return; // switched away
    currentText = comp.content ?? '';
    prevContent = comp.prev_content ?? null;
    undone = comp.undone ?? false;
    lastPromptTokens = comp.prompt_tokens ?? null;
    lastCompletionTokens = comp.completion_tokens ?? null;
    textareaEl.value = currentText;
    autoGrowTextarea();
    header.setSelection(comp);
    renderControls();
    updateUndoBtn();
    updateTokenStats();
  } catch (err) {
    console.error('failed to load completion:', err);
  }
}

async function newCompletion() {
  const sel = header.getSelection();
  const model = sel?.name ?? modelList[0]?.name ?? '';
  if (!model) return;
  try {
    const item = await completionsApi.create(model);
    sidebar.addItem(item);
    loadCompletion(item.id);
  } catch (err) {
    console.error('failed to create completion:', err);
  }
}

function handleRun() {
  if (!activeCompletionId || streaming || currentText.trim().length === 0) return;
  if (settleTimer) { clearTimeout(settleTimer); settleTimer = null; }

  prevContent = currentText;
  undone = false;
  undoBtnEl.classList.add('hidden');

  const promptText = currentText;
  genStart = promptText.length;
  settled = false;
  streaming = true;

  setMode('read');
  renderControls();

  updateUndoBtn();

  abortRun = completionsApi.run(activeCompletionId, promptText, {
    maxTokens,
    temperature,
    onDelta(delta) {
      currentText += delta;
      updateReadView();
      editorScrollEl.scrollTop = editorScrollEl.scrollHeight;
    },
    onDone() {
      finishStreaming();
    },
    onError(err) {
      console.error('completion run error:', err);
      finishStreaming();
    },
  });
}

async function finishStreaming() {
  streaming = false;
  abortRun = null;
  renderControls();
  updateReadView();
  updateUndoBtn();

  if (activeCompletionId) {
    try {
      const comp = await completionsApi.get(activeCompletionId);
      if (comp.id === activeCompletionId && comp.content != null) {
        currentText = comp.content;
        prevContent = comp.prev_content ?? null;
        undone = comp.undone ?? false;
        lastPromptTokens = comp.prompt_tokens ?? null;
        lastCompletionTokens = comp.completion_tokens ?? null;
        genStart = null;
        settled = true;
        textareaEl.value = currentText;
        autoGrowTextarea();
        updateReadView();
        updateTokenStats();
        return;
      }
    } catch (err) {
      console.error('failed to reload completion after run:', err);
    }
  }

  // Fallback if reload failed: fade the split view after a short delay.
  settleTimer = setTimeout(() => {
    settled = true;
    updateReadView();
    settleTimer = null;
  }, 900);
}

function handleStop() {
  if (abortRun) { abortRun(); abortRun = null; }
  finishStreaming();
}

function updateUndoBtn() {
  if (prevContent != null && !streaming) {
    undoBtnEl.innerHTML = undone
      ? icon('rotate-cw', 14) + ' Redo run'
      : icon('rotate-ccw', 14) + ' Undo run';
    undoBtnEl.classList.remove('hidden');
  } else {
    undoBtnEl.classList.add('hidden');
  }
}

async function handleUndo() {
  if (prevContent == null || undone) return;
  undoBtnEl.classList.add('hidden');
  try {
    const result = await completionsApi.undo(activeCompletionId);
    currentText = result.content;
    prevContent = textareaEl.value; // the post-run content is now the snapshot
    undone = true;
  } catch (err) {
    console.error('failed to undo completion:', err);
    return;
  }
  genStart = null;
  settled = true;
  textareaEl.value = currentText;
  setMode('write');
  renderControls();
  updateUndoBtn();
  setTimeout(() => { editorScrollEl.scrollTop = editorScrollEl.scrollHeight; }, 0);
}

async function handleRedo() {
  if (prevContent == null || !undone) return;
  undoBtnEl.classList.add('hidden');
  try {
    const result = await completionsApi.redo(activeCompletionId);
    currentText = result.content;
    prevContent = textareaEl.value; // the pre-run content is now the snapshot
    undone = false;
  } catch (err) {
    console.error('failed to redo completion:', err);
    return;
  }
  genStart = null;
  settled = true;
  textareaEl.value = currentText;
  setMode('write');
  renderControls();
  updateUndoBtn();
  setTimeout(() => { editorScrollEl.scrollTop = editorScrollEl.scrollHeight; }, 0);
}

function updateTokenStats() {
  header.setTokenStats(lastPromptTokens, lastCompletionTokens);
}

start();
