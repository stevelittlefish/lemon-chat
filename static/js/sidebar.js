import { conversations as api } from './api.js';
import { icon } from './icons.js';
import { escapeHtml } from './utils.js';

const sidebarEl = document.getElementById('sidebar');

let state = {
  items: [],
  activeId: null,
  onSelect: null,
  onNew: null,
  username: '',
};

// Body-level context menu — avoids overflow clipping from the scrollable list
let _menuEl = null;
let _menuConvId = null;

function openConvMenu(id, anchorEl) {
  if (_menuConvId === id) { closeConvMenu(); return; }
  closeConvMenu();
  const rect = anchorEl.getBoundingClientRect();
  const menu = document.createElement('div');
  menu.className = 'menu conv-context-menu';
  menu.style.cssText = `position:fixed;top:${rect.bottom + 4}px;right:${document.documentElement.clientWidth - rect.right}px;z-index:200;min-width:160px;`;
  menu.innerHTML = `
    <div class="menu-item" data-action="edit-title">
      ${icon('pencil', 14)} Edit title
    </div>
    <div class="menu-item" data-action="regen">
      ${icon('refresh-cw', 14)} Regenerate title
    </div>
    <div class="menu-sep"></div>
    <div class="menu-item danger" data-action="delete">
      ${icon('trash', 14)} Delete
    </div>
  `;
  document.body.appendChild(menu);
  _menuEl = menu;
  _menuConvId = id;

  menu.querySelector('[data-action="edit-title"]').addEventListener('click', (e) => {
    e.stopPropagation();
    closeConvMenu();
    startInlineEdit(id);
  });

  menu.querySelector('[data-action="regen"]').addEventListener('click', async (e) => {
    e.stopPropagation();
    closeConvMenu();
    await api.regenerateTitle(id);
  });

  menu.querySelector('[data-action="delete"]').addEventListener('click', (e) => {
    e.stopPropagation();
    closeConvMenu();
    showDeleteConfirm(id);
  });
}

function closeConvMenu() {
  if (_menuEl) {
    _menuEl.remove();
    _menuEl = null;
    _menuConvId = null;
  }
}

function startInlineEdit(id) {
  const itemEl = sidebarEl.querySelector(`.sidebar-item[data-id="${id}"]`);
  if (!itemEl) return;
  const titleEl = itemEl.querySelector('.sidebar-item-title');
  if (!titleEl) return;

  const conv = state.items.find(c => c.id === id);
  const current = conv?.title ?? '';

  const input = document.createElement('input');
  input.type = 'text';
  input.className = 'sidebar-item-title-input';
  input.value = current;

  let committed = false;
  function commit() {
    if (committed) return;
    committed = true;
    input.replaceWith(titleEl);
    const trimmed = input.value.trim();
    if (trimmed && trimmed !== current) {
      api.update(id, { title: trimmed }).then(() => updateTitle(id, trimmed));
    }
  }
  function cancel() {
    if (committed) return;
    committed = true;
    input.replaceWith(titleEl);
  }

  input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') { e.preventDefault(); commit(); }
    if (e.key === 'Escape') { e.preventDefault(); cancel(); }
  });
  input.addEventListener('blur', commit);
  input.addEventListener('mousedown', (e) => e.stopPropagation());
  input.addEventListener('click', (e) => e.stopPropagation());

  titleEl.replaceWith(input);
  input.focus();
  input.select();
}

let _deleteModal = null;

function getDeleteModal() {
  if (_deleteModal) return _deleteModal;

  const overlay = document.createElement('div');
  overlay.className = 'fork-modal';
  overlay.setAttribute('role', 'dialog');
  overlay.setAttribute('aria-modal', 'true');
  overlay.setAttribute('aria-label', 'Delete conversation');

  const dialog = document.createElement('div');
  dialog.className = 'fork-modal-dialog';

  const title = document.createElement('p');
  title.className = 'fork-modal-title';
  title.textContent = 'Delete conversation';

  const body = document.createElement('p');
  body.className = 'fork-modal-body';
  body.textContent = 'This conversation and all its messages will be permanently deleted.';

  const actions = document.createElement('div');
  actions.className = 'fork-modal-actions';

  const cancelBtn = document.createElement('button');
  cancelBtn.className = 'btn btn-secondary btn-sm';
  cancelBtn.textContent = 'Cancel';
  cancelBtn.addEventListener('click', closeDeleteModal);

  const confirmBtn = document.createElement('button');
  confirmBtn.className = 'btn btn-danger btn-sm';
  confirmBtn.textContent = 'Delete';

  actions.appendChild(cancelBtn);
  actions.appendChild(confirmBtn);
  dialog.appendChild(title);
  dialog.appendChild(body);
  dialog.appendChild(actions);
  overlay.appendChild(dialog);

  overlay.addEventListener('mousedown', (e) => {
    if (e.target === overlay) closeDeleteModal();
  });
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && overlay.classList.contains('open')) closeDeleteModal();
  });

  document.body.appendChild(overlay);
  _deleteModal = { overlay, confirmBtn };
  return _deleteModal;
}

function showDeleteConfirm(id) {
  const { overlay, confirmBtn } = getDeleteModal();
  confirmBtn.disabled = false;
  confirmBtn.textContent = 'Delete';
  confirmBtn.onclick = async () => {
    confirmBtn.disabled = true;
    confirmBtn.textContent = 'Deleting…';
    try {
      await api.delete(id);
      removeConversation(id);
      if (state.activeId === id) state.onSelect?.(null);
    } finally {
      closeDeleteModal();
    }
  };
  overlay.classList.add('open');
}

function closeDeleteModal() {
  _deleteModal?.overlay.classList.remove('open');
}

document.addEventListener('click', (e) => {
  if (_menuEl && !e.target.closest('.conv-context-menu') && !e.target.closest('.sidebar-item-menu')) {
    closeConvMenu();
  }
});

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
      <button class="btn btn-ghost btn-sm btn-icon" id="menu-btn" title="Menu">
        ${icon('layout-grid', 16)}
      </button>
      <button class="btn btn-ghost btn-sm btn-icon" id="settings-btn" title="Settings">
        ${icon('settings', 16)}
      </button>
      <button class="btn btn-ghost btn-sm btn-icon" id="logout-btn" title="Sign out">
        ${icon('log-out', 16)}
      </button>
    </div>
  `;

  document.getElementById('new-chat-btn').addEventListener('click', () => state.onNew?.());
  document.getElementById('menu-btn').addEventListener('click', () => { window.location.href = '/menu'; });
  document.getElementById('settings-btn').addEventListener('click', () => { window.location.href = '/settings/account'; });
  document.getElementById('logout-btn').addEventListener('click', handleLogout);

  sidebarEl.querySelectorAll('.sidebar-item').forEach(el => {
    const id = Number(el.dataset.id);
    el.addEventListener('click', (e) => {
      e.preventDefault();
      if (e.target.closest('.sidebar-item-menu')) return;
      state.onSelect?.(id);
    });
    el.querySelector('.sidebar-item-menu').addEventListener('click', (e) => {
      e.stopPropagation();
      e.preventDefault();
      openConvMenu(id, e.currentTarget);
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
      <button class="sidebar-item-menu" title="More options">${icon('ellipsis-vertical', 14)}</button>
    </a>
  `;
}

async function handleLogout() {
  const { auth } = await import('./api.js');
  await auth.logout();
  window.location.reload();
}

