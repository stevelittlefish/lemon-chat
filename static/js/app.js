import { auth, conversations as convApi, messages as msgApi, models as modelApi, characters as characterApi } from './api.js';
import * as sidebar from './sidebar.js';
import * as thread from './thread.js';
import { setBackground, resolveAttachment } from './thread.js';
import * as composer from './composer.js';
import * as header from './header.js';
import * as ws from './ws.js';
import { preload as preloadIcons } from './icons.js';

const loginScreen = document.getElementById('login-screen');
const appEl = document.getElementById('app');
const loginForm = document.getElementById('login-form');
const loginError = document.getElementById('login-error');
const composerContainer = document.getElementById('composer-container');

let currentUser = null;
let activeConversationId = null;
let activeHasMessages = false;
let modelList = [];
let characterList = [];
let currentAbort = null;

async function start() {
  try {
    currentUser = await auth.me();
    showApp();
  } catch {
    showLogin();
  }
}

function showLogin() {
  loginScreen.classList.remove('hidden');
  appEl.classList.add('hidden');
}

function showApp() {
  loginScreen.classList.add('hidden');
  appEl.classList.remove('hidden');
  ws.on('conversation_titled', ({ id, title }) => {
    sidebar.updateTitle(id, title);
    if (id === activeConversationId) header.updateTitle(title);
  });
  ws.on('conversations_changed', () => {
    sidebar.load();
  });
  ws.on('attachment_ready', (msg) => {
    resolveAttachment(msg.id, msg);
  });
  // A background was set on a conversation. Patch the cached item so re-loading
  // the conversation restores the background, and apply it now if it's active.
  ws.on('conversation_background', ({ id, attachment_id }) => {
    sidebar.updateItem(id, { background_attachment_id: attachment_id });
    if (id === activeConversationId) setBackground(attachment_id);
  });
  ws.on('attachment_error', (msg) => {
    resolveAttachment(msg.id, null);
  });
  ws.connect();
  window.addEventListener('popstate', (e) => {
    if (thread.closeLightboxIfOpen()) return;
    const id = e.state?.conversationId ?? null;
    loadConversation(id, { pushHistory: false });
  });
  initApp();
}

function getConversationIdFromUrl() {
  const params = new URLSearchParams(location.search);
  const c = params.get('c');
  return c ? Number(c) : null;
}

function showPickerScreen() {
  composerContainer.classList.add('hidden');
  thread.showPicker(modelList, characterList, handlePickerSelect);
}

async function handlePickerSelect(sel) {
  header.selectDirect(sel);

  if (sel.type === 'model') {
    composerContainer.classList.remove('hidden');
    thread.showEmpty();
    return;
  }

  // Character: create conversation so the first message can be shown.
  try {
    const conv = await convApi.create(null, null, sel.id);
    sidebar.addItem(conv);
    activeConversationId = conv.id;
    thread.setConversationId(conv.id);
    activeHasMessages = false;
    history.pushState({ conversationId: conv.id }, '', `/?c=${conv.id}`);
    header.setConversation(conv.id, conv.title ?? null);
    applyAvatarContext(sel.id);
    composerContainer.classList.remove('hidden');
    thread.showEmpty();
    await applyFirstMessage(conv.id, null);
  } catch {
    // Creation failed — stay on picker
  }
}

async function initApp() {
  await preloadIcons();

  sidebar.init({
    username: currentUser.username,
    onSelect: loadConversation,
    onNew: () => loadConversation(null),
    api: convApi,
    newLabel: 'New conversation',
    paginated: true,
  });

  let models, chars;
  try {
    [models, chars] = await Promise.all([
      modelApi.list('chat'),
      characterApi.list(),
      sidebar.load(),
    ]);
  } catch (err) {
    const threadEl = document.getElementById('thread');
    const msg = document.createElement('p');
    msg.className = 'init-error';
    msg.textContent = 'could not load — ' + (err.message || 'network error');
    threadEl.appendChild(msg);
    return;
  }
  modelList = models;
  characterList = chars;

  header.init({
    models: modelList,
    characters: characterList,
    onChange: handleSelectionChange,
  });

  composer.init({
    onSend: sendMessage,
    onStop: () => { if (currentAbort) currentAbort(); },
    onUpload: async (file) => {
      const convId = await ensureConversation();
      return msgApi.uploadAttachment(convId, file);
    },
  });
  updateCanAttach();

  thread.setForkHandler(async (messageId) => {
    if (!activeConversationId) return;
    const conv = await convApi.fork(activeConversationId, messageId);
    sidebar.addItem(conv);
    await loadConversation(conv.id);
  });

  const initialId = getConversationIdFromUrl();
  history.replaceState({ conversationId: initialId ?? null }, '', location.href);
  if (initialId) {
    await loadConversation(initialId, { pushHistory: false });
  } else {
    showPickerScreen();
  }
}

function applyAvatarContext(characterId) {
  const char = characterId ? characterList.find(c => c.id === characterId) : null;
  thread.setAvatarContext({
    userId: currentUser.id,
    userHasAvatar: currentUser.has_avatar,
    characterId: characterId ?? null,
    characterHasAvatar: char?.has_avatar ?? false,
    characterMap: new Map(characterList.map(c => [c.id, c.has_avatar])),
  });
}

async function loadConversation(id, { pushHistory = true } = {}) {
  thread.closeArtifactPanel();
  if (pushHistory) {
    const url = id ? `/?c=${id}` : '/';
    history.pushState({ conversationId: id ?? null }, '', url);
  }
  if (!id) {
    activeConversationId = null;
    activeHasMessages = false;
    header.setConversation(null, null);
    sidebar.setActive(null);
    setBackground(null);
    showPickerScreen();
    return;
  }
  composerContainer.classList.remove('hidden');
  activeConversationId = id;
  thread.setConversationId(id);
  sidebar.setActive(id);
  try {
    const msgs = await msgApi.list(id);
    activeHasMessages = msgs.length > 0;
    const conv = sidebar.getItem(id);
    header.setConversation(id, conv?.title ?? null);
    if (conv) header.setSelection(conv);
    updateCanAttach();
    applyAvatarContext(conv?.character_id ?? null);
    setBackground(conv?.background_attachment_id ?? null);
    thread.renderMessages(msgs);
  } catch {
    activeConversationId = null;
    thread.setConversationId(null);
    sidebar.setActive(null);
    history.replaceState({ conversationId: null }, '', '/');
    showPickerScreen();
  }
}

// selectionSupportsVision reports whether the current selection's model accepts
// image input, gating the composer's attach control.
function selectionSupportsVision(sel) {
  if (!sel) return false;
  let modelName = null;
  if (sel.type === 'model') modelName = sel.name;
  else if (sel.type === 'character') modelName = characterList.find(c => c.id === sel.id)?.model ?? null;
  if (!modelName) return false;
  return !!modelList.find(m => m.name === modelName)?.vision;
}

function updateCanAttach() {
  composer.setCanAttach(selectionSupportsVision(header.getSelection()));
}

// ensureConversation returns the active conversation id, creating a new
// conversation (and applying a character's first message) if none is active.
// Called both when sending and when attaching an image before the first send.
async function ensureConversation() {
  if (activeConversationId) return activeConversationId;
  const sel = header.getSelection();
  const model = sel?.type === 'model' ? sel.name : null;
  const charId = sel?.type === 'character' ? sel.id : null;
  const conv = await convApi.create(null, model, charId);
  sidebar.addItem(conv);
  activeConversationId = conv.id;
  thread.setConversationId(conv.id);
  activeHasMessages = false;
  history.pushState({ conversationId: conv.id }, '', `/?c=${conv.id}`);
  header.setConversation(conv.id, conv.title ?? null);
  applyAvatarContext(charId);
  if (charId !== null) {
    await applyFirstMessage(conv.id, null);
  }
  return conv.id;
}

async function sendMessage(content, attachments = []) {
  const sel = header.getSelection();
  try {
    await ensureConversation();
  } catch (err) {
    thread.appendMessage('user', content, currentUser.display_name || 'you', null, attachments);
    thread.startStreaming().error('could not send — ' + err.message);
    return;
  }

  const convId = activeConversationId;
  activeHasMessages = true;
  thread.appendMessage('user', content, currentUser.display_name || 'you', null, attachments);
  let stream = thread.startStreaming();
  composer.setStreaming(true);
  let streamHasToolCalls = false;
  let currentTurnName = null;

  currentAbort = msgApi.send(convId, content, sel, {
    onName: (name) => {
      currentTurnName = name;
      stream.setName(name);
    },
    onDelta: (delta) => {
      if (streamHasToolCalls) {
        stream.finish();
        stream = thread.startStreaming();
        if (currentTurnName) stream.setName(currentTurnName);
        streamHasToolCalls = false;
      }
      stream.append(delta);
    },
    onStats: (stats) => stream.setStats(stats),
    onMessageId: (id) => stream.setMessageId(id),
    onToolCall: (tc) => {
      streamHasToolCalls = true;
      stream.addToolCall(tc);
    },
    onToolResult: (tr) => stream.updateToolCallResult(tr),
    onAttachment: (att) => stream.addAttachment(att),
    onNewTurn: (info) => {
      if (info?.name) currentTurnName = info.name;
    },
    onDone: () => {
      stream.finish();
      streamHasToolCalls = false;
      composer.setStreaming(false);
      currentAbort = null;
      sidebar.updateItem(convId, {
        model: sel?.type === 'model' ? sel.name : null,
        character_id: sel?.type === 'character' ? sel.id : null,
      });
    },
    onAborted: () => {
      stream.truncate();
      streamHasToolCalls = false;
      composer.setStreaming(false);
      currentAbort = null;
    },
    onError: (err) => {
      stream.error('something went wrong — ' + err.message);
      streamHasToolCalls = false;
      composer.setStreaming(false);
      currentAbort = null;
    },
  }, attachments.map(a => a.id));
}

// Called when the picker selection changes. If the conversation is empty and
// the new selection is a character, save and show their first message.
async function handleSelectionChange(sel) {
  updateCanAttach();
  if (!activeConversationId) return;
  applyAvatarContext(sel?.type === 'character' ? sel.id : null);
  if (activeHasMessages) return;
  if (!sel || sel.type !== 'character') return;
  await applyFirstMessage(activeConversationId, sel.id);
}

// Saves and displays the first message for a character.
// Pass charId to override (and update) the conversation's character; pass null
// to use the character already set on the conversation.
async function applyFirstMessage(convId, charId) {
  try {
    const msg = await msgApi.firstMessage(convId, charId);
    if (!msg) return;
    thread.appendMessage('assistant', msg.content, msg.name, msg.character_id);
    activeHasMessages = true;
    if (charId !== null) {
      sidebar.updateItem(convId, { character_id: charId, model: null });
    }
  } catch (err) {
    // 422: character has no first message / no character set — expected, silent
    // 409: conversation already has messages — expected, silent
    if (err.status === 422 || err.status === 409) return;
    thread.startStreaming().error('could not load first message — ' + (err.message || 'network error'));
  }
}

// Login form
loginForm.addEventListener('submit', async (e) => {
  e.preventDefault();
  loginError.classList.add('hidden');
  const username = loginForm.username.value.trim();
  const password = loginForm.password.value;
  try {
    currentUser = await auth.login(username, password);
    showApp();
  } catch (err) {
    loginError.textContent = err.message;
    loginError.classList.remove('hidden');
  }
});

start();
