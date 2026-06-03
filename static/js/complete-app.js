import { auth, models as modelApi, completions as completionsApi } from './api.js';
import * as sidebar from './sidebar.js';
import * as header from './header.js';
import { preload as preloadIcons } from './icons.js';

const appEl = document.getElementById('app');
const completeEmptyEl = document.getElementById('complete-empty');

let currentUser = null;
let activeCompletionId = null;
let modelList = [];

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
    onChange: null,
  });

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
}

function showEmpty() {
  activeCompletionId = null;
  completeEmptyEl.classList.remove('hidden');
  header.setConversation(null, null);
  sidebar.setActive(null);
}

function loadCompletion(id, { pushHistory = true } = {}) {
  if (!id) {
    if (pushHistory) history.pushState({ completionId: null }, '', '/complete');
    showEmpty();
    return;
  }
  if (pushHistory) history.pushState({ completionId: id }, '', `/complete?c=${id}`);
  activeCompletionId = id;
  completeEmptyEl.classList.add('hidden');
  sidebar.setActive(id);
  const item = sidebar.getItem(id);
  header.setConversation(id, item?.title ?? null);
  // TODO: load and display completion content
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

start();
