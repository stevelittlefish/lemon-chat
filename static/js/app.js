import { auth, conversations as convApi, messages as msgApi, models as modelApi, characters as characterApi } from './api.js';
import * as sidebar from './sidebar.js';
import * as thread from './thread.js';
import * as composer from './composer.js';
import * as header from './header.js';
import * as ws from './ws.js';
import { preload as preloadIcons } from './icons.js';

const loginScreen = document.getElementById('login-screen');
const appEl = document.getElementById('app');
const loginForm = document.getElementById('login-form');
const loginError = document.getElementById('login-error');

let currentUser = null;
let activeConversationId = null;
let activeHasMessages = false;

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
  ws.connect();
  initApp();
}

async function initApp() {
  const [modelList, characterList] = await Promise.all([
    modelApi.list(),
    characterApi.list(),
    sidebar.load(),
    preloadIcons(),
  ]);

  sidebar.init({
    username: currentUser.username,
    onSelect: loadConversation,
    onNew: newConversation,
  });

  header.init({
    models: modelList,
    characters: characterList,
    onChange: handleSelectionChange,
  });

  composer.init({
    onSend: sendMessage,
  });

  thread.showEmpty();
}

async function loadConversation(id) {
  if (!id) {
    activeConversationId = null;
    activeHasMessages = false;
    header.setConversation(null, null);
    thread.showEmpty();
    sidebar.setActive(null);
    return;
  }
  activeConversationId = id;
  sidebar.setActive(id);
  const msgs = await msgApi.list(id);
  activeHasMessages = msgs.length > 0;
  const conv = sidebar.getConversation(id);
  header.setConversation(id, conv?.title ?? null);
  if (conv) header.setSelection(conv);
  thread.renderMessages(msgs);
}

async function newConversation() {
  const sel = header.getSelection();
  const model = sel?.type === 'model' ? sel.name : null;
  const charId = sel?.type === 'character' ? sel.id : null;
  const conv = await convApi.create(null, model, charId);
  sidebar.addConversation(conv);
  activeConversationId = conv.id;
  activeHasMessages = false;
  header.setConversation(conv.id, conv.title ?? null);
  thread.showEmpty();
  if (charId !== null) {
    await applyFirstMessage(conv.id, null);
  }
}

async function sendMessage(content) {
  const sel = header.getSelection();
  if (!activeConversationId) {
    const model = sel?.type === 'model' ? sel.name : null;
    const charId = sel?.type === 'character' ? sel.id : null;
    const conv = await convApi.create(null, model, charId);
    sidebar.addConversation(conv);
    activeConversationId = conv.id;
    activeHasMessages = false;
    header.setConversation(conv.id, conv.title ?? null);
    if (charId !== null) {
      await applyFirstMessage(conv.id, null);
    }
  }

  const convId = activeConversationId;
  activeHasMessages = true;
  thread.appendMessage('user', content);
  const stream = thread.startStreaming();
  composer.setStreaming(true);

  msgApi.send(convId, content, sel, {
    onName: (name) => stream.setName(name),
    onDelta: (delta) => stream.append(delta),
    onDone: () => {
      stream.finish();
      composer.setStreaming(false);
      sidebar.updateConversation(convId, {
        model: sel?.type === 'model' ? sel.name : null,
        character_id: sel?.type === 'character' ? sel.id : null,
      });
    },
    onError: (err) => {
      stream.error('something went wrong — ' + err.message);
      composer.setStreaming(false);
    },
  });
}

// Called when the picker selection changes. If the conversation is empty and
// the new selection is a character, save and show their first message.
async function handleSelectionChange(sel) {
  if (!activeConversationId) return;
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
    thread.appendMessage('assistant', msg.content, msg.name);
    activeHasMessages = true;
    if (charId !== null) {
      sidebar.updateConversation(convId, { character_id: charId, model: null });
    }
  } catch {
    // Character has no first message, or already has messages — silent
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
