import { conversations as api } from './api.js';

const sidebarEl = document.getElementById('sidebar');

let state = {
  items: [],
  activeId: null,
  onSelect: null,
  onNew: null,
  username: '',
};

export function init({ onSelect, onNew, username }) {
  state.onSelect = onSelect;
  state.onNew = onNew;
  state.username = username;
  render();
}

export async function load() {
  state.items = await api.list();
  render();
}

export function setActive(id) {
  state.activeId = id;
  render();
}

export function addConversation(conv) {
  state.items.unshift(conv);
  state.activeId = conv.id;
  render();
}

export function removeConversation(id) {
  state.items = state.items.filter(c => c.id !== id);
  if (state.activeId === id) state.activeId = null;
  render();
}

function render() {
  sidebarEl.innerHTML = `
    <div class="sidebar-header">
      <img src="/assets/logo-wordmark.svg" alt="lemon chat" class="sidebar-logo">
      <button class="btn btn-ghost btn-sm btn-icon" id="new-chat-btn" title="New conversation">
        ${iconPlus()}
      </button>
    </div>
    <div class="sidebar-list" id="sidebar-list">
      ${state.items.map(convItem).join('')}
    </div>
    <div class="sidebar-footer">
      <span class="sidebar-user">${escapeHtml(state.username)}</span>
      <button class="btn btn-ghost btn-sm btn-icon" id="logout-btn" title="Sign out">
        ${iconLogOut()}
      </button>
    </div>
  `;

  document.getElementById('new-chat-btn').addEventListener('click', () => state.onNew?.());
  document.getElementById('logout-btn').addEventListener('click', handleLogout);

  sidebarEl.querySelectorAll('.sidebar-item').forEach(el => {
    const id = Number(el.dataset.id);
    el.addEventListener('click', (e) => {
      if (e.target.closest('.sidebar-item-delete')) return;
      state.onSelect?.(id);
    });
    el.querySelector('.sidebar-item-delete').addEventListener('click', async () => {
      if (!confirm('Delete this conversation?')) return;
      await api.delete(id);
      removeConversation(id);
      if (state.activeId === id) state.onSelect?.(null);
    });
  });
}

function convItem(conv) {
  const active = conv.id === state.activeId ? ' active' : '';
  const titleHtml = conv.title
    ? `<span class="sidebar-item-title">${escapeHtml(conv.title)}</span>`
    : `<span class="sidebar-item-title sidebar-item-title--empty">(new conversation)</span>`;
  return `
    <div class="sidebar-item${active}" data-id="${conv.id}">
      ${titleHtml}
      <button class="sidebar-item-delete" title="Delete">${iconTrash()}</button>
    </div>
  `;
}

async function handleLogout() {
  const { auth } = await import('./api.js');
  await auth.logout();
  window.location.reload();
}

function escapeHtml(str) {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function iconPlus() {
  return `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>`;
}

function iconTrash() {
  return `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14H6L5 6"/><path d="M10 11v6M14 11v6"/><path d="M9 6V4h6v2"/></svg>`;
}

function iconLogOut() {
  return `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>`;
}
