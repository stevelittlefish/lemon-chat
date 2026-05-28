import { auth, characters as charactersApi, models as modelsApi } from './api.js';
import { preload as preloadIcons, icon } from './icons.js';

let user = null;
let svgArrowLeft, svgUser, svgUsers, svgCpu, svgPlus, svgPencil, svgTrash;

let charactersData = [];
let modelsData     = [];
let editingCharId  = null;
let showingCharAddForm = false;

async function init() {
  try {
    user = await auth.me();
  } catch {
    window.location.href = '/';
    return;
  }
  await preloadIcons();
  svgArrowLeft = icon('arrow-left', 14);
  svgUser      = icon('user', 16);
  svgUsers     = icon('users', 16);
  svgCpu       = icon('drama', 16);
  svgPlus      = icon('plus', 14);
  svgPencil    = icon('pencil', 13);
  svgTrash     = icon('trash', 13);
  renderNav();
  renderPage();
}

// ── Nav ───────────────────────────────────────────────────────────────

function renderNav() {
  const path = window.location.pathname;
  const adminSection = user.is_admin ? `
    <div class="snav-group-label">Admin</div>
    <a href="/settings/users" class="snav-item${path === '/settings/users' ? ' active' : ''}">
      ${svgUsers}
      Users
    </a>
  ` : '';

  document.getElementById('snav').innerHTML = `
    <a href="/" class="snav-back">
      ${svgArrowLeft}
      Back to chat
    </a>
    <div class="snav-head">
      <img src="/assets/logo-mark.svg" alt="">
      <div class="title">Settings</div>
    </div>
    <div class="snav-group-label">You</div>
    <a href="/settings/account" class="snav-item${path === '/settings/account' ? ' active' : ''}">
      ${svgUser}
      Account
    </a>
    <a href="/settings/characters" class="snav-item${path === '/settings/characters' ? ' active' : ''}">
      ${svgCpu}
      Characters
    </a>
    ${adminSection}
  `;
}

// ── Characters page ───────────────────────────────────────────────────

async function renderPage() {
  editingCharId      = null;
  showingCharAddForm = false;

  document.getElementById('smain').innerHTML = `
    <div class="smain-head">
      <h1>Characters</h1>
      <p>Reusable AI personas with a model, system prompt, and optional opening message.</p>
    </div>
    <div class="section">
      <div class="section-toolbar">
        <div>
          <h2>My characters</h2>
          <p class="lead">Characters you created.</p>
        </div>
        <button class="btn btn-secondary btn-sm" id="add-char-btn">
          ${svgPlus}
          Add character
        </button>
      </div>
      <table class="chars-table">
        <thead><tr><th>Name</th><th>Model</th><th>Visibility</th><th></th></tr></thead>
        <tbody id="my-chars-list"><tr><td colspan="4" class="users-status">Loading…</td></tr></tbody>
      </table>
    </div>
    <div class="section" id="other-chars-section">
      <h2>Other characters</h2>
      <p class="lead">Characters created by other users.</p>
      <table class="chars-table">
        <thead><tr><th>Name</th><th>Model</th><th>Visibility</th><th></th></tr></thead>
        <tbody id="other-chars-list"><tr><td colspan="4" class="users-status">Loading…</td></tr></tbody>
      </table>
    </div>
  `;

  document.getElementById('add-char-btn').addEventListener('click', () => {
    showingCharAddForm = true;
    editingCharId = null;
    renderCharacterLists();
  });

  try {
    [charactersData, modelsData] = await Promise.all([
      charactersApi.list(),
      modelsApi.list(),
    ]);
  } catch {
    const err = `<div class="field-msg field-msg--error" style="padding:14px 0">Could not load data.</div>`;
    document.getElementById('my-chars-list').innerHTML = err;
    document.getElementById('other-chars-list').innerHTML = err;
    return;
  }
  renderCharacterLists();
}

function canEditChar(c) {
  return c.created_by === user.id || user.is_admin || c.visibility === 'readwrite';
}

function canDeleteChar(c) {
  return c.created_by === user.id || user.is_admin;
}

function renderCharacterLists() {
  const myList       = document.getElementById('my-chars-list');
  const otherList    = document.getElementById('other-chars-list');
  const otherSection = document.getElementById('other-chars-section');
  if (!myList) return;

  const mine  = charactersData.filter(c => c.created_by === user.id);
  const other = charactersData.filter(c => c.created_by !== user.id);

  myList.innerHTML    = renderCharRows(mine, true);
  otherList.innerHTML = renderCharRows(other, false);
  if (otherSection) otherSection.hidden = other.length === 0;

  attachCharEvents(myList);
  attachCharEvents(otherList);

  if (showingCharAddForm) {
    const input = document.getElementById('char-add-name');
    if (input) {
      input.focus();
      input.addEventListener('keydown', e => {
        if (e.key === 'Escape') { showingCharAddForm = false; renderCharacterLists(); }
      });
    }
  }

  if (editingCharId !== null) {
    const input = document.getElementById(`char-edit-name-${editingCharId}`);
    if (input) {
      input.focus();
      input.select();
      input.addEventListener('keydown', e => {
        if (e.key === 'Escape') { editingCharId = null; renderCharacterLists(); }
      });
    }
  }
}

function modelOptions(selected) {
  return modelsData.map(m =>
    `<option value="${escapeHtml(m.name)}" ${m.name === selected ? 'selected' : ''}>${escapeHtml(m.display_name || m.name)}</option>`
  ).join('');
}

function charFormHtml(idPrefix, c) {
  const name         = c ? escapeHtml(c.name)             : '';
  const model        = c ? c.model                         : (modelsData[0]?.name ?? '');
  const systemPrompt = c ? (c.system_prompt ?? '')         : '';
  const firstMessage = c ? (c.first_message ?? '')         : '';
  const visibility   = c ? c.visibility                    : 'private';
  const isOwnerOrAdmin = !c || c.created_by === user.id || user.is_admin;

  return `
    <tr class="character-row--form" id="${idPrefix}-form">
      <td colspan="4">
        <div class="character-form-fields">
          <div class="character-form-row">
            <label class="character-form-lbl" for="${idPrefix}-name">Name</label>
            <input id="${idPrefix}-name" class="input" value="${name}" placeholder="Character name" autocomplete="off">
          </div>
          <div class="character-form-row">
            <label class="character-form-lbl" for="${idPrefix}-model">Model</label>
            <select id="${idPrefix}-model" class="input">${modelOptions(model)}</select>
          </div>
          <div class="character-form-row">
            <label class="character-form-lbl" for="${idPrefix}-system-prompt">System prompt</label>
            <textarea id="${idPrefix}-system-prompt" class="input" rows="4" placeholder="Optional system prompt…">${escapeHtml(systemPrompt)}</textarea>
          </div>
          <div class="character-form-row">
            <label class="character-form-lbl" for="${idPrefix}-first-message">First message</label>
            <textarea id="${idPrefix}-first-message" class="input" rows="3" placeholder="Optional opening message…">${escapeHtml(firstMessage)}</textarea>
          </div>
          ${isOwnerOrAdmin ? `
          <div class="character-form-row">
            <label class="character-form-lbl" for="${idPrefix}-visibility">Visibility</label>
            <select id="${idPrefix}-visibility" class="input">
              <option value="private"   ${visibility === 'private'   ? 'selected' : ''}>Private — only you</option>
              <option value="readonly"  ${visibility === 'readonly'  ? 'selected' : ''}>Read-only — others can see, not edit</option>
              <option value="readwrite" ${visibility === 'readwrite' ? 'selected' : ''}>Read-write — others can see and edit</option>
            </select>
          </div>` : ''}
        </div>
        <div class="field-msg field-msg--error" id="${idPrefix}-err" hidden></div>
        <div class="user-row-actions">
          <button class="btn btn-ghost btn-sm" data-action="${c ? 'cancel-edit' : 'cancel-add'}" ${c ? `data-id="${c.id}"` : ''}>Cancel</button>
          <button class="btn btn-primary btn-sm" data-action="${c ? 'save-edit' : 'save-add'}" ${c ? `data-id="${c.id}"` : ''}>${c ? 'Save' : 'Add character'}</button>
        </div>
      </td>
    </tr>`;
}

function renderCharRows(list, isMine) {
  let html = '';

  if (isMine && showingCharAddForm) {
    html += charFormHtml('char-add', null);
  }

  if (!list.length && !(isMine && showingCharAddForm)) {
    html += `<tr><td colspan="4" class="users-status">${isMine ? 'You have no characters yet.' : 'No characters from other users.'}</td></tr>`;
  }

  for (const c of list) {
    if (editingCharId === c.id) {
      html += charFormHtml(`char-edit-${c.id}`, c);
    } else {
      const editBtn   = canEditChar(c)   ? `<button class="btn btn-ghost btn-sm" data-action="edit" data-id="${c.id}">${svgPencil} Edit</button>` : '';
      const deleteBtn = canDeleteChar(c) ? `<button class="btn btn-ghost btn-sm" data-action="delete" data-id="${c.id}">${svgTrash} Delete</button>` : '';
      const visLabel  = { private: 'private', readonly: 'read-only', readwrite: 'read-write' }[c.visibility] ?? c.visibility;
      html += `
        <tr data-id="${c.id}">
          <td class="char-col-name">${escapeHtml(c.name)}</td>
          <td class="char-col-model"><span class="chip">${escapeHtml(c.model)}</span></td>
          <td class="char-col-visibility"><span class="chip character-chip--${c.visibility}">${visLabel}</span></td>
          <td class="char-col-actions"><div class="user-row-actions">${editBtn}${deleteBtn}</div></td>
        </tr>`;
    }
  }

  return html;
}

function attachCharEvents(container) {
  container.querySelectorAll('[data-action]').forEach(btn => {
    const action = btn.dataset.action;
    const id     = Number(btn.dataset.id);
    switch (action) {
      case 'edit':
        btn.addEventListener('click', () => { editingCharId = id; showingCharAddForm = false; renderCharacterLists(); });
        break;
      case 'cancel-edit':
        btn.addEventListener('click', () => { editingCharId = null; renderCharacterLists(); });
        break;
      case 'save-edit':
        btn.addEventListener('click', () => saveEditChar(id));
        break;
      case 'cancel-add':
        btn.addEventListener('click', () => { showingCharAddForm = false; renderCharacterLists(); });
        break;
      case 'save-add':
        btn.addEventListener('click', () => createChar());
        break;
      case 'delete':
        btn.addEventListener('click', () => deleteChar(id));
        break;
    }
  });
}

function readCharForm(prefix) {
  const isOwnerOrAdmin = prefix === 'char-add' || (() => {
    const id = Number(prefix.replace('char-edit-', ''));
    const c  = charactersData.find(c => c.id === id);
    return c && (c.created_by === user.id || user.is_admin);
  })();
  return {
    name:          document.getElementById(`${prefix}-name`)?.value.trim()          ?? '',
    model:         document.getElementById(`${prefix}-model`)?.value                ?? '',
    system_prompt: document.getElementById(`${prefix}-system-prompt`)?.value.trim() || null,
    first_message: document.getElementById(`${prefix}-first-message`)?.value.trim() || null,
    visibility:    isOwnerOrAdmin ? (document.getElementById(`${prefix}-visibility`)?.value ?? undefined) : undefined,
  };
}

async function createChar() {
  const prefix  = 'char-add';
  const data    = readCharForm(prefix);
  const errorEl = document.getElementById(`${prefix}-err`);
  const saveBtn = document.querySelector(`[data-action="save-add"]`);

  errorEl.hidden = true;
  if (!data.name)  { errorEl.hidden = false; errorEl.textContent = 'Name is required.';  return; }
  if (!data.model) { errorEl.hidden = false; errorEl.textContent = 'Model is required.'; return; }

  saveBtn.disabled    = true;
  saveBtn.textContent = 'Adding…';

  try {
    const created = await charactersApi.create(data);
    charactersData.push(created);
    charactersData.sort((a, b) => a.name.localeCompare(b.name));
    showingCharAddForm = false;
    renderCharacterLists();
  } catch (err) {
    saveBtn.disabled    = false;
    saveBtn.textContent = 'Add character';
    errorEl.hidden      = false;
    errorEl.textContent = err.message;
  }
}

async function saveEditChar(id) {
  const prefix  = `char-edit-${id}`;
  const data    = readCharForm(prefix);
  const errorEl = document.getElementById(`${prefix}-err`);
  const saveBtn = document.querySelector(`[data-action="save-edit"][data-id="${id}"]`);

  errorEl.hidden = true;
  if (!data.name)  { errorEl.hidden = false; errorEl.textContent = 'Name is required.';  return; }
  if (!data.model) { errorEl.hidden = false; errorEl.textContent = 'Model is required.'; return; }

  saveBtn.disabled    = true;
  saveBtn.textContent = 'Saving…';

  try {
    await charactersApi.update(id, data);
    const c = charactersData.find(c => c.id === id);
    if (c) {
      c.name          = data.name;
      c.model         = data.model;
      c.system_prompt = data.system_prompt;
      c.first_message = data.first_message;
      if (data.visibility !== undefined) c.visibility = data.visibility;
    }
    charactersData.sort((a, b) => a.name.localeCompare(b.name));
    editingCharId = null;
    renderCharacterLists();
  } catch (err) {
    saveBtn.disabled    = false;
    saveBtn.textContent = 'Save';
    errorEl.hidden      = false;
    errorEl.textContent = err.message;
  }
}

async function deleteChar(id) {
  const c = charactersData.find(c => c.id === id);
  if (!confirm(`Delete character "${c?.name}"? This cannot be undone.`)) return;

  try {
    await charactersApi.delete(id);
    charactersData = charactersData.filter(c => c.id !== id);
    if (editingCharId === id) editingCharId = null;
    renderCharacterLists();
  } catch (err) {
    alert(`Could not delete character: ${err.message}`);
  }
}

// ── Helpers ───────────────────────────────────────────────────────────

function escapeHtml(str) {
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

init();
