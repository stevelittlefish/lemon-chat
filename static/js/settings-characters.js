import { auth, characters as charactersApi } from './api.js';
import { preload as preloadIcons, icon } from './icons.js';

let user = null;
let svgArrowLeft, svgUser, svgUsers, svgCpu, svgPlus, svgPencil, svgTrash;

let charactersData = [];

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
        <a href="/settings/characters/new" class="btn btn-secondary btn-sm">
          ${svgPlus}
          Add character
        </a>
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

  try {
    charactersData = await charactersApi.list();
  } catch {
    const err = `<div class="field-msg field-msg--error" style="padding:14px 0">Could not load characters.</div>`;
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

  myList.innerHTML    = renderCharRows(mine,  'You have no characters yet.');
  otherList.innerHTML = renderCharRows(other, 'No characters from other users.');
  if (otherSection) otherSection.hidden = other.length === 0;

  attachDeleteEvents(myList);
  attachDeleteEvents(otherList);
}

function renderCharRows(list, emptyMsg) {
  if (!list.length) {
    return `<tr><td colspan="4" class="users-status">${emptyMsg}</td></tr>`;
  }
  return list.map(c => {
    const editBtn   = canEditChar(c)   ? `<a href="/settings/characters/${c.id}/edit" class="btn btn-ghost btn-sm">${svgPencil} Edit</a>` : '';
    const deleteBtn = canDeleteChar(c) ? `<button class="btn btn-ghost btn-sm" data-action="delete" data-id="${c.id}">${svgTrash} Delete</button>` : '';
    const visLabel  = { private: 'private', readonly: 'read-only', readwrite: 'read-write' }[c.visibility] ?? c.visibility;
    return `
      <tr data-id="${c.id}">
        <td class="char-col-name">${escapeHtml(c.name)}</td>
        <td class="char-col-model"><span class="chip">${escapeHtml(c.model)}</span></td>
        <td class="char-col-visibility"><span class="chip character-chip--${c.visibility}">${visLabel}</span></td>
        <td class="char-col-actions"><div class="user-row-actions">${editBtn}${deleteBtn}</div></td>
      </tr>`;
  }).join('');
}

function attachDeleteEvents(container) {
  container.querySelectorAll('[data-action="delete"]').forEach(btn => {
    const id = Number(btn.dataset.id);
    btn.addEventListener('click', () => deleteChar(id));
  });
}

async function deleteChar(id) {
  const c = charactersData.find(c => c.id === id);
  if (!confirm(`Delete character "${c?.name}"? This cannot be undone.`)) return;

  try {
    await charactersApi.delete(id);
    charactersData = charactersData.filter(c => c.id !== id);
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
