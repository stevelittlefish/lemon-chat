import { conversations as api } from './api.js';
import { icon } from './icons.js';

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

export function getConversation(id) {
  return state.items.find(c => c.id === id) ?? null;
}

export function updateConversation(id, updates) {
  const idx = state.items.findIndex(c => c.id === id);
  if (idx !== -1) state.items[idx] = { ...state.items[idx], ...updates };
}

export function updateTitle(id, title) {
  const idx = state.items.findIndex(c => c.id === id);
  if (idx === -1) return;
  state.items[idx] = { ...state.items[idx], title };
  const el = sidebarEl.querySelector(`.sidebar-item[data-id="${id}"] .sidebar-item-title`);
  if (el) {
    el.textContent = title;
    el.classList.remove('sidebar-item-title--empty');
  }
}

function render() {
  sidebarEl.innerHTML = `
    <div class="sidebar-header">
      <div class="sidebar-brand">
        <img src="/assets/logo-mark.svg" alt="" class="sidebar-logo">
        <span class="sidebar-brand-name">lemon chat</span>
      </div>
      <button class="btn btn-ghost btn-sm btn-icon" id="new-chat-btn" title="New conversation">
        ${icon('plus', 18)}
      </button>
    </div>
    <div class="sidebar-list" id="sidebar-list">
      ${state.items.map(convItem).join('')}
    </div>
    <div class="sidebar-footer">
      <span class="sidebar-user">${escapeHtml(state.username)}</span>
      <button class="btn btn-ghost btn-sm btn-icon" id="settings-btn" title="Settings">
        ${icon('settings', 16)}
      </button>
      <button class="btn btn-ghost btn-sm btn-icon" id="logout-btn" title="Sign out">
        ${icon('log-out', 16)}
      </button>
    </div>
  `;

  document.getElementById('new-chat-btn').addEventListener('click', () => state.onNew?.());
  document.getElementById('settings-btn').addEventListener('click', () => { window.location.href = '/settings/account'; });
  document.getElementById('logout-btn').addEventListener('click', handleLogout);

  sidebarEl.querySelectorAll('.sidebar-item').forEach(el => {
    const id = Number(el.dataset.id);
    el.addEventListener('click', (e) => {
      e.preventDefault();
      if (e.target.closest('.sidebar-item-delete')) return;
      state.onSelect?.(id);
    });
    el.querySelector('.sidebar-item-delete').addEventListener('click', async (e) => {
      e.stopPropagation();
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
    <a class="sidebar-item${active}" data-id="${conv.id}" href="/?c=${conv.id}">
      ${titleHtml}
      <button class="sidebar-item-delete" title="Delete">${icon('trash', 14)}</button>
    </a>
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

