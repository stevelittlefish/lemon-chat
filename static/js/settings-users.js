import { auth, admin } from './api.js';
import { preload as preloadIcons, icon } from './icons.js';

let user = null;
let svgArrowLeft, svgUser, svgUsers, svgCpu, svgPlus, svgPencil, svgTrash;

let usersData      = [];
let editingId      = null;
let showingAddForm = false;

async function init() {
  try {
    user = await auth.me();
  } catch {
    window.location.href = '/';
    return;
  }
  if (!user.is_admin) {
    window.location.href = '/settings/account';
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
    <div class="snav-group-label">Admin</div>
    <a href="/settings/users" class="snav-item${path === '/settings/users' ? ' active' : ''}">
      ${svgUsers}
      Users
    </a>
  `;
}

// ── Users page ────────────────────────────────────────────────────────

async function renderPage() {
  editingId      = null;
  showingAddForm = false;

  document.getElementById('smain').innerHTML = `
    <div class="smain-head">
      <h1>Users</h1>
      <p>Everyone who can sign in to this server.</p>
    </div>
    <div class="section">
      <div class="section-toolbar">
        <div>
          <h2>All users</h2>
          <p class="lead">Add or edit accounts. New users are created without a password.</p>
        </div>
        <button class="btn btn-secondary btn-sm" id="add-user-btn">
          ${svgPlus}
          Add user
        </button>
      </div>
      <div id="users-list"><div class="users-status">Loading…</div></div>
    </div>
  `;

  document.getElementById('add-user-btn').addEventListener('click', () => {
    showingAddForm = true;
    editingId = null;
    renderUsersList();
  });

  try {
    usersData = await admin.users.list();
  } catch {
    document.getElementById('users-list').innerHTML =
      `<div class="field-msg field-msg--error" style="padding:14px 0">Could not load users.</div>`;
    return;
  }
  renderUsersList();
}

function renderUsersList() {
  const list = document.getElementById('users-list');
  if (!list) return;

  let html = '';

  if (showingAddForm) {
    html += `
      <div class="user-row user-row--form">
        <input class="input user-name-input" id="add-username" placeholder="username" autocomplete="off" spellcheck="false">
        <label class="user-admin-check">
          <input type="checkbox" id="add-is-admin"> Admin
        </label>
        <div class="user-row-actions">
          <button class="btn btn-ghost btn-sm" id="cancel-add-btn">Cancel</button>
          <button class="btn btn-primary btn-sm" id="save-add-btn">Add user</button>
        </div>
      </div>
      <div class="user-form-err field-msg field-msg--error" id="add-err" hidden></div>`;
  }

  if (!usersData.length && !showingAddForm) {
    html += '<div class="users-status">No users yet.</div>';
  }

  for (const u of usersData) {
    if (editingId === u.id) {
      html += `
        <div class="user-row user-row--form" data-id="${u.id}">
          <input class="input user-name-input" id="edit-username-${u.id}" value="${escapeHtml(u.username)}" autocomplete="off" spellcheck="false">
          <label class="user-admin-check">
            <input type="checkbox" id="edit-is-admin-${u.id}" ${u.is_admin ? 'checked' : ''}> Admin
          </label>
          <div class="user-row-actions">
            <button class="btn btn-ghost btn-sm" data-action="cancel-edit">Cancel</button>
            <button class="btn btn-primary btn-sm" data-action="save-edit" data-id="${u.id}">Save</button>
          </div>
        </div>
        <div class="user-form-err field-msg field-msg--error" id="edit-err-${u.id}" hidden></div>`;
    } else {
      html += `
        <div class="user-row" data-id="${u.id}">
          <div class="user-name">${escapeHtml(u.username)}</div>
          ${u.is_admin ? '<span class="user-badge">admin</span>' : '<span></span>'}
          <div class="user-row-actions">
            <button class="btn btn-ghost btn-sm" data-action="edit" data-id="${u.id}">${svgPencil} Edit</button>
            <button class="btn btn-ghost btn-sm" data-action="delete" data-id="${u.id}">${svgTrash} Delete</button>
          </div>
        </div>`;
    }
  }

  list.innerHTML = html;

  if (showingAddForm) {
    const usernameInput = document.getElementById('add-username');
    document.getElementById('cancel-add-btn').addEventListener('click', () => {
      showingAddForm = false;
      renderUsersList();
    });
    document.getElementById('save-add-btn').addEventListener('click', createUser);
    usernameInput.addEventListener('keydown', e => {
      if (e.key === 'Enter') createUser();
      if (e.key === 'Escape') { showingAddForm = false; renderUsersList(); }
    });
    usernameInput.focus();
  }

  list.querySelectorAll('[data-action]').forEach(btn => {
    const action = btn.dataset.action;
    const id = Number(btn.dataset.id);
    switch (action) {
      case 'edit':
        btn.addEventListener('click', () => { editingId = id; showingAddForm = false; renderUsersList(); });
        break;
      case 'cancel-edit':
        btn.addEventListener('click', () => { editingId = null; renderUsersList(); });
        break;
      case 'save-edit':
        btn.addEventListener('click', () => saveEditUser(id));
        break;
      case 'delete':
        btn.addEventListener('click', () => deleteUser(id));
        break;
    }
  });

  if (editingId !== null) {
    const input = document.getElementById(`edit-username-${editingId}`);
    if (input) {
      input.focus();
      input.select();
      input.addEventListener('keydown', e => {
        if (e.key === 'Enter') saveEditUser(editingId);
        if (e.key === 'Escape') { editingId = null; renderUsersList(); }
      });
    }
  }
}

async function createUser() {
  const usernameInput = document.getElementById('add-username');
  const username = usernameInput.value.trim();
  const isAdmin  = document.getElementById('add-is-admin').checked;
  const errorEl  = document.getElementById('add-err');
  const saveBtn  = document.getElementById('save-add-btn');

  errorEl.hidden = true;
  if (!username) {
    errorEl.hidden = false;
    errorEl.textContent = 'Username is required.';
    usernameInput.focus();
    return;
  }

  saveBtn.disabled    = true;
  saveBtn.textContent = 'Adding…';

  try {
    const newUser = await admin.users.create({ username, is_admin: isAdmin });
    usersData.push(newUser);
    usersData.sort((a, b) => a.username.localeCompare(b.username));
    showingAddForm = false;
    renderUsersList();
  } catch (err) {
    saveBtn.disabled    = false;
    saveBtn.textContent = 'Add user';
    errorEl.hidden      = false;
    errorEl.textContent = err.message;
  }
}

async function saveEditUser(id) {
  const usernameInput = document.getElementById(`edit-username-${id}`);
  const username = usernameInput.value.trim();
  const isAdmin  = document.getElementById(`edit-is-admin-${id}`).checked;
  const errorEl  = document.getElementById(`edit-err-${id}`);
  const saveBtn  = document.querySelector(`[data-action="save-edit"][data-id="${id}"]`);

  errorEl.hidden = true;
  if (!username) {
    errorEl.hidden = false;
    errorEl.textContent = 'Username is required.';
    usernameInput.focus();
    return;
  }

  saveBtn.disabled    = true;
  saveBtn.textContent = 'Saving…';

  try {
    await admin.users.update(id, { username, is_admin: isAdmin });
    const u = usersData.find(u => u.id === id);
    if (u) { u.username = username; u.is_admin = isAdmin; }
    usersData.sort((a, b) => a.username.localeCompare(b.username));
    editingId = null;
    renderUsersList();
  } catch (err) {
    saveBtn.disabled    = false;
    saveBtn.textContent = 'Save';
    errorEl.hidden      = false;
    errorEl.textContent = err.message;
  }
}

async function deleteUser(id) {
  const u = usersData.find(u => u.id === id);
  if (!confirm(`Delete user "${u?.username}"? This cannot be undone.`)) return;

  try {
    await admin.users.delete(id);
    usersData = usersData.filter(u => u.id !== id);
    if (editingId === id) editingId = null;
    renderUsersList();
  } catch (err) {
    alert(`Could not delete user: ${err.message}`);
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
