import { notes as notesApi } from './api.js';
import { requireAuth } from './settings-auth.js';
import { preload as preloadIcons, icon } from './icons.js';
import { escapeHtml } from './utils.js';

let user = null;
let notesData = [];
let editingId = null;
let showingAddForm = false;

async function init() {
  user = await requireAuth();
  if (!user) return;
  await preloadIcons();
  renderNav();
  await loadNotes();
  renderPage();
}

// ── Nav ───────────────────────────────────────────────────────────────

function renderNav() {
  const path = window.location.pathname;
  const adminSection = user.is_admin ? `
    <div class="snav-group-label">Admin</div>
    <a href="/settings/users" class="snav-item${path === '/settings/users' ? ' active' : ''}">
      ${icon('users', 16)}
      Users
    </a>
    <a href="/settings/tools" class="snav-item${path === '/settings/tools' ? ' active' : ''}">
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
    <a href="/settings/account" class="snav-item${path === '/settings/account' ? ' active' : ''}">
      ${icon('user', 16)}
      Account
    </a>
    <a href="/settings/characters" class="snav-item${path === '/settings/characters' ? ' active' : ''}">
      ${icon('drama', 16)}
      Characters
    </a>
    <a href="/settings/notes" class="snav-item${path === '/settings/notes' ? ' active' : ''}">
      ${icon('file-text', 16)}
      Notes
    </a>
    ${adminSection}
  `;
}

// ── Data ──────────────────────────────────────────────────────────────

async function loadNotes() {
  notesData = await notesApi.list();
}

// ── Render ────────────────────────────────────────────────────────────

function scopeLabel(key) {
  switch (key[0]) {
    case 'g': return 'global';
    case 'u': return 'user';
    case 'c': return 'conversation';
    default:  return '';
  }
}

function renderNoteRow(note) {
  const scope = scopeLabel(note.key);
  const ro = note.read_only
    ? `<span class="note-badge note-badge--ro">read-only</span>`
    : '';
  const canEdit = !note.read_only && note.key[0] !== 'c';
  const editBtn = canEdit
    ? `<button class="btn btn-ghost btn-sm" data-action="edit" data-id="${note.id}">${icon('pencil', 13)}</button>`
    : '';
  return `
    <div class="note-row${editingId === note.id ? ' note-row--editing' : ''}" data-id="${note.id}">
      <div class="note-row-main">
        <div class="note-row-key">
          <code>${escapeHtml(note.key)}</code>
          <span class="note-scope-tag note-scope-tag--${scope}">${scope}</span>
          ${ro}
        </div>
        <div class="note-row-excerpt">${escapeHtml(note.excerpt)}${note.excerpt.length >= 120 ? '…' : ''}</div>
      </div>
      <div class="note-row-actions">
        <button class="btn btn-ghost btn-sm" data-action="toggle-ro" data-id="${note.id}" data-ro="${note.read_only}" title="${note.read_only ? 'Unlock' : 'Lock'}">
          ${icon(note.read_only ? 'lock' : 'eye', 13)}
        </button>
        ${editBtn}
        <button class="btn btn-ghost btn-sm note-delete-btn" data-action="delete" data-id="${note.id}" data-key="${escapeHtml(note.key)}" ${note.read_only ? 'disabled' : ''}>${icon('trash', 13)}</button>
      </div>
    </div>
    ${editingId === note.id ? renderEditForm(note) : ''}
  `;
}

function renderEditForm(note) {
  return `
    <form class="note-edit-form" data-id="${note.id}">
      <textarea class="input note-value-input" rows="8" name="value">${escapeHtml(note.value)}</textarea>
      <div class="note-form-actions">
        <button type="submit" class="btn btn-primary btn-sm">Save</button>
        <button type="button" class="btn btn-ghost btn-sm" data-action="cancel-edit">Cancel</button>
        <span class="field-msg" id="edit-msg-${note.id}"></span>
      </div>
    </form>
  `;
}

function renderAddForm() {
  return `
    <form class="note-add-form section" id="note-add-form">
      <h2>New note</h2>
      <div class="note-key-row">
        <select class="input note-scope-select" name="scope">
          <option value="g">g. — global</option>
          <option value="u">u. — user</option>
        </select>
        <input class="input note-key-input" type="text" name="keyname"
          placeholder="eldoria.bestiary"
          pattern="[a-z0-9_-]+(\.[a-z0-9_-]+)*"
          title="Lowercase letters, digits, underscores, hyphens, dots as separators. No consecutive dots.">
      </div>
      <textarea class="input note-value-input" rows="8" name="value" placeholder="Note content…"></textarea>
      <label class="note-ro-label">
        <input type="checkbox" name="read_only"> Lock as read-only
      </label>
      <div class="note-form-actions">
        <button type="submit" class="btn btn-primary btn-sm">Create note</button>
        <button type="button" class="btn btn-ghost btn-sm" id="cancel-add-btn">Cancel</button>
        <span class="field-msg" id="add-msg"></span>
      </div>
    </form>
  `;
}

function renderPage() {
  const smain = document.getElementById('smain');

  const grouped = { g: [], u: [], c: [] };
  for (const n of notesData) {
    const scope = n.key[0];
    if (grouped[scope]) grouped[scope].push(n);
  }

  const renderGroup = (label, notes) => {
    if (notes.length === 0) return '';
    return `
      <div class="note-group">
        <div class="note-group-label">${label}</div>
        ${notes.map(renderNoteRow).join('')}
      </div>
    `;
  };

  const anyNotes = notesData.length > 0;

  smain.innerHTML = `
    <div class="smain-head">
      <h1>Notes</h1>
      <p>Long-form notes for lorebooks, session briefs, and memories. Global notes are visible to all users and characters.</p>
    </div>
    <div class="section">
      <div class="section-toolbar">
        <div>
          <h2>All notes</h2>
        </div>
        <button class="btn btn-primary btn-sm" id="add-note-btn">${icon('plus', 14)} New note</button>
      </div>
      ${showingAddForm ? renderAddForm() : ''}
      ${anyNotes
        ? `<div class="notes-list">
            ${renderGroup('Global', grouped.g)}
            ${renderGroup('User', grouped.u)}
            ${renderGroup('Conversation', grouped.c)}
          </div>`
        : `<p class="lead" style="margin-top: 16px">No notes yet. Create one to get started.</p>`
      }
    </div>
  `;

  attachHandlers();
}

function attachHandlers() {
  const addBtn = document.getElementById('add-note-btn');
  if (addBtn) {
    addBtn.addEventListener('click', () => {
      showingAddForm = true;
      editingId = null;
      renderPage();
    });
  }

  const cancelAddBtn = document.getElementById('cancel-add-btn');
  if (cancelAddBtn) {
    cancelAddBtn.addEventListener('click', () => {
      showingAddForm = false;
      renderPage();
    });
  }

  const addForm = document.getElementById('note-add-form');
  if (addForm) {
    addForm.addEventListener('submit', handleAddSubmit);
  }

  document.querySelectorAll('[data-action="edit"]').forEach(btn => {
    btn.addEventListener('click', () => {
      editingId = parseInt(btn.dataset.id, 10);
      showingAddForm = false;
      renderPage();
      document.querySelector(`.note-edit-form[data-id="${editingId}"] textarea`)?.focus();
    });
  });

  document.querySelectorAll('[data-action="cancel-edit"]').forEach(btn => {
    btn.addEventListener('click', () => {
      editingId = null;
      renderPage();
    });
  });

  document.querySelectorAll('.note-edit-form').forEach(form => {
    form.addEventListener('submit', handleEditSubmit);
  });

  document.querySelectorAll('[data-action="delete"]').forEach(btn => {
    btn.addEventListener('click', () => handleDelete(parseInt(btn.dataset.id, 10), btn.dataset.key));
  });

  document.querySelectorAll('[data-action="toggle-ro"]').forEach(btn => {
    btn.addEventListener('click', () => handleToggleReadOnly(parseInt(btn.dataset.id, 10), btn.dataset.ro === 'true'));
  });
}

// ── Actions ───────────────────────────────────────────────────────────

async function handleAddSubmit(e) {
  e.preventDefault();
  const form = e.target;
  const msg  = document.getElementById('add-msg');
  const scope   = form.elements.scope.value;
  const keyname = form.elements.keyname.value.trim();
  const value   = form.elements.value.value;
  const readOnly = form.elements.read_only.checked;

  if (!keyname) { showMsg(msg, 'error', 'Key name is required.'); return; }
  const key = `${scope}.${keyname}`;

  try {
    const created = await notesApi.upsert(key, value, readOnly);
    notesData.push(created);
    showingAddForm = false;
    editingId = null;
    renderPage();
  } catch (err) {
    showMsg(msg, 'error', err.message || 'Failed to create note.');
  }
}

async function handleEditSubmit(e) {
  e.preventDefault();
  const form  = e.target;
  const id    = parseInt(form.dataset.id, 10);
  const value = form.elements.value.value;
  const msg   = document.getElementById(`edit-msg-${id}`);
  const note  = notesData.find(n => n.id === id);

  try {
    const updated = await notesApi.upsert(note.key, value);
    const idx = notesData.findIndex(n => n.id === id);
    if (idx !== -1) {
      notesData[idx] = { ...notesData[idx], excerpt: updated.value.slice(0, 120), ...updated };
    }
    editingId = null;
    renderPage();
  } catch (err) {
    showMsg(msg, 'error', err.message || 'Failed to save note.');
  }
}

async function handleDelete(id, key) {
  if (!confirm(`Delete note "${key}"?\n\nThis cannot be undone.`)) return;
  try {
    await notesApi.delete(id);
    notesData = notesData.filter(n => n.id !== id);
    if (editingId === id) editingId = null;
    renderPage();
  } catch (err) {
    alert('Failed to delete note: ' + (err.message || 'unknown error'));
  }
}

async function handleToggleReadOnly(id, currentlyReadOnly) {
  const newVal = !currentlyReadOnly;
  try {
    const updated = await notesApi.setReadOnly(id, newVal);
    const idx = notesData.findIndex(n => n.id === id);
    if (idx !== -1) notesData[idx] = { ...notesData[idx], read_only: updated.read_only };
    renderPage();
  } catch (err) {
    alert('Failed to update read-only: ' + (err.message || 'unknown error'));
  }
}

function showMsg(el, type, text) {
  if (!el) return;
  el.className = `field-msg field-msg--${type === 'error' ? 'error' : 'success'}`;
  el.textContent = text;
}

init();
