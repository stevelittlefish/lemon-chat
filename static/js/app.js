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

  header.init({ models: modelList, characters: characterList });

  composer.init({
    onSend: sendMessage,
  });

  thread.showEmpty();
}

async function loadConversation(id) {
  if (!id) {
    activeConversationId = null;
    header.setConversation(null, null);
    thread.showEmpty();
    sidebar.setActive(null);
    return;
  }
  activeConversationId = id;
  sidebar.setActive(id);
  const msgs = await msgApi.list(id);
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
  header.setConversation(conv.id, conv.title ?? null);
  thread.showEmpty();
}

async function sendMessage(content) {
  const sel = header.getSelection();
  if (!activeConversationId) {
    const model = sel?.type === 'model' ? sel.name : null;
    const charId = sel?.type === 'character' ? sel.id : null;
    const conv = await convApi.create(null, model, charId);
    sidebar.addConversation(conv);
    activeConversationId = conv.id;
    header.setConversation(conv.id, conv.title ?? null);
  }

  const convId = activeConversationId;
  thread.appendMessage('user', content);
  const stream = thread.startStreaming();
  composer.setStreaming(true);

  msgApi.send(convId, content, sel, {
    onName: (name) => stream.setName(name),
    onDelta: (delta) => stream.append(delta),
    onDone: () => {
      stream.finish();
      composer.setStreaming(false);
      // Keep sidebar cache in sync with the model/character actually used.
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
