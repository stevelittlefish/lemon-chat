import { auth, conversations as convApi, messages as msgApi, models as modelApi } from './api.js';
import * as sidebar from './sidebar.js';
import * as thread from './thread.js';
import * as composer from './composer.js';

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
  initApp();
}

async function initApp() {
  const [modelList] = await Promise.all([
    modelApi.list(),
    sidebar.load(),
  ]);

  sidebar.init({
    username: currentUser.username,
    onSelect: loadConversation,
    onNew: newConversation,
  });

  composer.init({
    models: modelList,
    onSend: sendMessage,
  });

  thread.showEmpty();
}

async function loadConversation(id) {
  if (!id) {
    activeConversationId = null;
    thread.showEmpty();
    sidebar.setActive(null);
    return;
  }
  activeConversationId = id;
  sidebar.setActive(id);
  const msgs = await msgApi.list(id);
  thread.renderMessages(msgs);
}

async function newConversation() {
  const conv = await convApi.create(null);
  sidebar.addConversation(conv);
  activeConversationId = conv.id;
  thread.showEmpty();
}

async function sendMessage(content, model) {
  if (!activeConversationId) {
    const conv = await convApi.create(null);
    sidebar.addConversation(conv);
    activeConversationId = conv.id;
  }

  thread.appendMessage('user', content);
  const stream = thread.startStreaming();
  composer.setStreaming(true);

  msgApi.send(activeConversationId, content, model, {
    onDelta: (delta) => stream.append(delta),
    onDone: () => {
      stream.finish();
      composer.setStreaming(false);
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
