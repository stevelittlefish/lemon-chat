import { memories as memoriesApi, admin as adminApi } from './api.js';
import { requireAuth } from './settings-auth.js';
import { preload as preloadIcons, icon } from './icons.js';
import { escapeHtml } from './utils.js';

// Matches memoryMaxChars in internal/server/tools_memory.go.
const MAX_CHARS = 500;

let user = null;
let users = [];        // admins only: everyone, for the user picker
let ownerId = null;    // whose memories are shown
let memoriesData = [];
let editingId = null;

async function init() {
  user = await requireAuth();
  if (!user) return;
  await preloadIcons();
  ownerId = user.id;
  renderNav();
  if (user.is_admin) {
    try { users = await adminApi.users.list(); } catch { users = []; }
  }
  await loadMemories();
}

// ── Nav ───────────────────────────────────────────────────────────────

function renderNav() {
  const adminSection = user.is_admin ? `
    <div class="snav-group-label">Admin</div>
    <a href="/settings/users" class="snav-item">
      ${icon('users', 16)}
      Users
    </a>
    <a href="/settings/tools" class="snav-item">
      ${icon('sliders', 16)}
      Tools
    </a>
  ` : '';

  document.getElementById('snav').innerHTML = `
    <a href="/" class="snav-back">
      ${icon('arrow-left', 14)}
      Back to chat
    </a>
    <div class="snav-head">
      <img src="/assets/logo-mark.svg" alt="">
      <div class="title">Settings</div>
    </div>
    <div class="snav-group-label">You</div>
    <a href="/settings/account" class="snav-item">
      ${icon('user', 16)}
      Account
    </a>
    <a href="/settings/characters" class="snav-item">
      ${icon('drama', 16)}
      Characters
    </a>
    <a href="/settings/memories" class="snav-item active">
      ${icon('brain', 16)}
      Memories
    </a>
    ${adminSection}
  `;
}

// ── Data ──────────────────────────────────────────────────────────────

const otherUser = () => ownerId === user.id ? undefined : ownerId;

async function loadMemories() {
  try {
    memoriesData = await memoriesApi.list(otherUser());
  } catch (err) {
    memoriesData = [];
    renderPage(err.message || 'Could not load memories.');
    return;
  }
  editingId = null;
  renderPage();
}

// ── Render ────────────────────────────────────────────────────────────

function userLabel(u) {
  return u.display_name ? `${u.display_name} (${u.username})` : u.username;
}

function formatDate(iso) {
  const d = new Date(iso);
  return isNaN(d) ? '' : d.toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' });
}

function renderSource(m) {
  if (m.conversation_id == null) return 'Added here';
  // Only the owner can open the conversation.
  const title = escapeHtml(m.conversation_title || 'Untitled conversation');
  return ownerId === user.id ? `From <a href="/?c=${m.conversation_id}">${title}</a>` : `From ${title}`;
}

function renderRow(m) {
  if (editingId === m.id) {
    return `
      <form class="memory-row memory-edit-form" data-id="${m.id}">
        <textarea class="input memory-text-input" name="text" rows="3" maxlength="${MAX_CHARS}">${escapeHtml(m.text)}</textarea>
        <div class="memory-form-actions">
          <button type="submit" class="btn btn-primary btn-sm">Save</button>
          <button type="button" class="btn btn-ghost btn-sm" data-action="cancel-edit">Cancel</button>
          <span class="field-msg" id="edit-msg-${m.id}"></span>
        </div>
      </form>
    `;
  }
  // Only admins can change a protected memory, or protect one.
  const canEdit = user.is_admin || !m.protected;
  const protectedMeta = m.protected
    ? ` · <span class="memory-protected">${icon('lock', 12)} Protected${user.is_admin ? '' : ' — only an admin can change this'}</span>`
    : '';
  const protectBtn = user.is_admin
    ? `<button class="btn btn-ghost btn-sm" data-action="protect" data-id="${m.id}" title="${m.protected ? 'Unprotect' : 'Protect, so only an admin can change it'}">${icon(m.protected ? 'lock' : 'lock-open', 13)}</button>`
    : '';
  return `
    <div class="memory-row" data-id="${m.id}">
      <div class="memory-row-main">
        <div class="memory-row-text">${escapeHtml(m.text)}</div>
        <div class="memory-row-meta">${formatDate(m.updated_at)} · ${renderSource(m)}${protectedMeta}</div>
      </div>
      ${canEdit ? `
      <div class="memory-row-actions">
        ${protectBtn}
        <button class="btn btn-ghost btn-sm" data-action="edit" data-id="${m.id}" title="Edit">${icon('pencil', 13)}</button>
        <button class="btn btn-ghost btn-sm" data-action="delete" data-id="${m.id}" title="Delete">${icon('trash', 13)}</button>
      </div>` : ''}
    </div>
  `;
}

function renderPicker() {
  if (!user.is_admin || users.length < 2) return '';
  return `
    <div class="memory-picker">
      <label class="character-form-lbl" for="memory-user">Showing memories for</label>
      <select id="memory-user" class="input">
        ${users.map(u => `<option value="${u.id}" ${u.id === ownerId ? 'selected' : ''}>${escapeHtml(userLabel(u))}${u.id === user.id ? ' — you' : ''}</option>`).join('')}
      </select>
    </div>
  `;
}

function renderPage(loadError) {
  const whose = ownerId === user.id ? 'you' : userLabel(users.find(u => u.id === ownerId) ?? { username: 'this user' });
  document.getElementById('smain').innerHTML = `
    <div class="smain-head">
      <h1>Memories</h1>
      <p>What characters with the memory tool remember about ${escapeHtml(whose)} between conversations.${user.is_admin ? ' Protected memories can only be changed or deleted by an admin.' : ''}</p>
    </div>
    <div class="section">
      ${renderPicker()}
      <form class="memory-add-form" id="memory-add-form">
        <textarea class="input memory-text-input" name="text" rows="2" maxlength="${MAX_CHARS}" placeholder="Add a memory, e.g. Has a cat called Biscuit"></textarea>
        <div class="memory-form-actions">
          <button type="submit" class="btn btn-primary btn-sm">${icon('plus', 14)} Add memory</button>
          ${user.is_admin ? `<label class="memory-protect-check"><input type="checkbox" name="protected"> Protected</label>` : ''}
          <span class="field-msg" id="add-msg"></span>
        </div>
      </form>
      ${loadError
        ? `<div class="field-msg field-msg--error">${escapeHtml(loadError)}</div>`
        : memoriesData.length
          ? `<div class="memory-list">${memoriesData.map(renderRow).join('')}</div>`
          : `<p class="lead memory-empty">No memories yet.</p>`}
    </div>
  `;
  attachHandlers();
}

// ── Handlers ──────────────────────────────────────────────────────────

function attachHandlers() {
  document.getElementById('memory-user')?.addEventListener('change', e => {
    ownerId = Number(e.target.value);
    loadMemories();
  });

  document.getElementById('memory-add-form').addEventListener('submit', handleAdd);

  document.querySelectorAll('[data-action="edit"]').forEach(btn => {
    btn.addEventListener('click', () => {
      editingId = Number(btn.dataset.id);
      renderPage();
      const ta = document.querySelector(`.memory-edit-form[data-id="${editingId}"] textarea`);
      ta?.focus();
      ta?.setSelectionRange(ta.value.length, ta.value.length);
    });
  });
  document.querySelector('[data-action="cancel-edit"]')?.addEventListener('click', () => {
    editingId = null;
    renderPage();
  });
  document.querySelector('.memory-edit-form')?.addEventListener('submit', handleEdit);

  document.querySelectorAll('[data-action="delete"]').forEach(btn => {
    btn.addEventListener('click', () => handleDelete(Number(btn.dataset.id)));
  });
  document.querySelectorAll('[data-action="protect"]').forEach(btn => {
    btn.addEventListener('click', () => handleProtect(Number(btn.dataset.id)));
  });
}

async function handleAdd(e) {
  e.preventDefault();
  const text = e.target.elements.text.value.trim();
  const msg = document.getElementById('add-msg');
  if (!text) return;
  try {
    const isProtected = e.target.elements.protected?.checked ?? false;
    memoriesData.push(await memoriesApi.create(text, otherUser(), isProtected));
    renderPage();
  } catch (err) {
    showMsg(msg, err.message || 'Could not add the memory.');
  }
}

async function handleEdit(e) {
  e.preventDefault();
  const id = Number(e.target.dataset.id);
  const text = e.target.elements.text.value.trim();
  try {
    replaceMemory(await memoriesApi.update(id, { text }));
    editingId = null;
    renderPage();
  } catch (err) {
    showMsg(document.getElementById(`edit-msg-${id}`), err.message || 'Could not save the memory.');
  }
}

async function handleProtect(id) {
  const m = memoriesData.find(m => m.id === id);
  if (!m) return;
  try {
    replaceMemory(await memoriesApi.update(id, { protected: !m.protected }));
    renderPage();
  } catch (err) {
    window.alert('Could not change the memory: ' + (err.message || 'unknown error'));
  }
}

// replaceMemory swaps in an updated memory. The update response has no
// conversation title, so keep the one we have.
function replaceMemory(updated) {
  const i = memoriesData.findIndex(m => m.id === updated.id);
  if (i !== -1) memoriesData[i] = { ...memoriesData[i], ...updated };
}

async function handleDelete(id) {
  const m = memoriesData.find(m => m.id === id);
  if (!m || !window.confirm(`Delete this memory?\n\n${m.text}`)) return;
  try {
    await memoriesApi.delete(id);
    memoriesData = memoriesData.filter(m => m.id !== id);
    renderPage();
  } catch (err) {
    window.alert('Could not delete the memory: ' + (err.message || 'unknown error'));
  }
}

function showMsg(el, text) {
  if (!el) return;
  el.className = 'field-msg field-msg--error';
  el.textContent = text;
}

init();
